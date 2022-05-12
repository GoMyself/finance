package model

import (
	"errors"
	"finance/contrib/helper"
	"finance/contrib/validator"
	"fmt"

	g "github.com/doug-martin/goqu/v9"
	"github.com/valyala/fasthttp"
)

// Manual 调用与pid对应的渠道, 发起充值(代付)请求
func ManualPay(fctx *fasthttp.RequestCtx, payment_id, amount string) (map[string]string, error) {

	res := map[string]string{}
	user, err := MemberCache(fctx)
	if err != nil {
		return res, err
	}

	ts := fctx.Time().In(loc).Unix()
	p, err := CachePayment(payment_id)
	if err != nil {
		return res, errors.New(helper.ChannelNotExist)
	}

	// 检查存款金额是否符合范围
	a, ok := validator.CheckFloatScope(amount, p.Fmin, p.Fmax)
	if !ok {
		return res, errors.New(helper.AmountOutRange)
	}

	// 检查用户的存款行为是否过于频繁
	err = cacheDepositProcessing(user.UID, ts)
	if err != nil {
		return res, err
	}

	amount = a.Truncate(0).String()

	bc, err := BankCardBackend()
	if err != nil {
		fmt.Println("BankCardBackend err = ", err.Error())
		return res, errors.New(helper.BankCardNotExist)
	}

	// 获取附言码
	code, err := TransacCodeGet()
	if err != nil {
		return res, errors.New(helper.ChannelBusyTryOthers)
	}

	fmt.Println("TransacCodeGet code = ", code)

	// 生成我方存款订单号
	orderId := helper.GenId()

	d := g.Record{
		"id":            orderId,
		"prefix":        meta.Prefix,
		"oid":           orderId,
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
		"created_at":    ts,
		"created_uid":   "0",
		"created_name":  "",
		"confirm_at":    "0",
		"confirm_uid":   "0",
		"confirm_name":  "",
		"review_remark": "",
		"manual_remark": fmt.Sprintf(`{"manual_remark": "%s", "real_name":"%s", "bank_addr":"%s", "name":"%s"}`, code, bc.AccountName, bc.BankcardAddr, bc.BanklcardName),
		"bankcard_id":   bc.Id,
		"flag":          "3",
		//"bank_code":     bankCode,
		"bank_no": bc.BanklcardNo,
		"level":   user.Level,
	}

	// 请求成功插入订单
	err = deposit(d)
	if err != nil {
		fmt.Println("Manual deposit err = ", err)
		return res, errors.New(helper.DBErr)
	}

	// 记录存款行为
	_ = cacheDepositProcessingInsert(user.UID, orderId, fctx.Time().In(loc).Unix())

	res = map[string]string{
		"id":           orderId,
		"name":         bc.BanklcardName,
		"cardNo":       bc.BanklcardNo,
		"realname":     bc.AccountName,
		"bankAddr":     bc.BankcardAddr,
		"manualRemark": code,
	}

	return res, nil
}
