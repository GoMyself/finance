package model

import (
	"errors"
	"finance/contrib/helper"
	"finance/contrib/validator"
	"fmt"
	"time"

	g "github.com/doug-martin/goqu/v9"
	"github.com/valyala/fasthttp"
)

// Manual 调用与pid对应的渠道, 发起充值(代付)请求
func ManualPay(fctx *fasthttp.RequestCtx, pid, amount, bankcardID, bankCode string) (map[string]string, error) {

	res := map[string]string{}
	user, err := MemberCache(fctx)
	if err != nil {
		return res, err
	}

	pLog := &paymentTDLog{
		Lable:    paymentLogTag,
		Flag:     "deposit",
		Username: user.Username,
	}
	// 记录请求日志
	defer func() {
		if err != nil {
			pLog.Error = fmt.Sprintf("{req: %s, err: %s}", fctx.PostArgs().String(), err.Error())
		}
		paymentPushLog(pLog)
	}()

	p, err := CachePayment(pid)
	if err != nil {
		return res, errors.New(helper.ChannelNotExist)
	}

	ch, err := ChannelTypeById(p.ChannelID)
	if err != nil {
		fmt.Println("Manual ChannelTypeById = ", err.Error())
		return res, err
	}

	pLog.Merchant = "线下转卡"
	pLog.Channel = ch["name"]

	// 检查存款金额是否符合范围
	a, ok := validator.CheckFloatScope(amount, p.Fmin, p.Fmax)
	if !ok {
		return res, errors.New(helper.AmountOutRange)
	}

	// 检查用户的存款行为是否过于频繁
	err = cacheDepositProcessing(user.UID, time.Now().Unix())
	if err != nil {
		return res, err
	}

	// 获取银行卡
	card, err := BankCards(bankcardID)
	if err != nil {
		return res, errors.New(helper.ChannelBusyTryOthers)
	}

	// 获取附言码
	code, err := DepositManualRemark(bankcardID)
	if err != nil {
		return res, errors.New(helper.ChannelBusyTryOthers)
	}

	amount = a.Truncate(0).String()

	// 生成我方存款订单号
	pLog.OrderID = helper.GenId()

	d := g.Record{
		"id":            pLog.OrderID,
		"prefix":        meta.Prefix,
		"oid":           pLog.OrderID,
		"uid":           user.UID,
		"top_uid":       user.TopUID,
		"top_name":      user.TopName,
		"parent_name":   user.ParentName,
		"parent_uid":    user.ParentUID,
		"username":      user.Username,
		"channel_id":    p.ChannelID,
		"cid":           p.CateID,
		"pid":           p.ID,
		"amount":        amount,
		"state":         DepositConfirming,
		"finance_type":  TransactionOfflineDeposit,
		"automatic":     "0",
		"created_at":    fctx.Time().In(loc).Unix(),
		"created_uid":   "0",
		"created_name":  "",
		"confirm_at":    "0",
		"confirm_uid":   "0",
		"confirm_name":  "",
		"review_remark": "",
		"manual_remark": fmt.Sprintf(`{"manual_remark": "%s", "real_name":"%s", "bank_addr":"%s", "name":"%s"}`, code, card.RealName, card.BankAddr, card.Name),
		"bankcard_id":   card.ID,
		"flag":          DepositFlagManual,
		"bank_code":     bankCode,
		"bank_no":       card.CardNo,
		"level":         user.Level,
	}

	// 请求成功插入订单
	err = deposit(d)
	if err != nil {
		fmt.Println("Manual deposit err = ", err)
		return res, errors.New(helper.DBErr)
	}

	// 记录存款行为
	_ = cacheDepositProcessingInsert(user.UID, pLog.OrderID, fctx.Time().Unix())

	res = map[string]string{
		"id":           pLog.OrderID,
		"name":         card.Name,
		"cardNo":       card.CardNo,
		"realname":     card.RealName,
		"bankAddr":     card.BankAddr,
		"manualRemark": code,
	}

	return res, nil
}
