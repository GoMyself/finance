package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"
	"time"

	g "github.com/doug-martin/goqu/v9"
	"github.com/go-redis/redis/v8"
	"github.com/shopspring/decimal"
	"github.com/valyala/fasthttp"
)

func UsdtUpdate(field, value string) error {

	query := fmt.Sprintf(`insert into f_config(name, content,prefix) values ('%s', '%s','%s') on duplicate key update name = '%s', content = '%s',prefix= '%s'`, field, value, meta.Prefix, field, value, meta.Prefix)
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return errors.New(helper.DBErr)
	}

	err = meta.MerchantRedis.HSet(ctx, "usdt", field, value).Err()
	if err != nil {
		return errors.New(helper.RedisErr)
	}

	return nil
}

func UsdtInfo() (map[string]string, error) {

	res := map[string]string{}
	f, err := meta.MerchantRedis.HMGet(ctx, "usdt", "usdt_rate", "usdt_trc_addr").Result()
	if err != nil && redis.Nil != err {
		return res, errors.New(helper.RedisErr)
	}

	rate := ""
	addr := ""

	if v, ok := f[0].(string); ok {
		rate = v
	}
	if v, ok := f[1].(string); ok {
		addr = v
	}

	res["usdt_rate"] = rate
	res["usdt_trc_addr"] = addr

	return res, nil
}

// USDT 线下USDT支付
func UsdtPay(fctx *fasthttp.RequestCtx, pid, amount, addr, protocolType, hashID string) (string, error) {

	user, err := MemberCache(fctx)
	if err != nil {
		return "", err
	}

	/*
		pLog := &paymentTDLog{
			Lable:    paymentLogTag,
			Flag:     "deposit",
			Username: user.Username,
		}
		// 记录请求日志
		defer func() {
			if err != nil {
				pLog.Error = fmt.Sprintf("{req: %s, err: %s, user: %s}", fctx.PostArgs().String(), err.Error(), user.Username)
			}
			paymentPushLog(pLog)
		}()
	*/

	p, err := CachePayment(pid)
	if err != nil {
		return "", errors.New(helper.ChannelNotExist)
	}

	//ch := paymentChannelMatch(p.ChannelID)
	/*
		ch, err := ChannelTypeById(p.ChannelID)
		if err != nil {
			return "", errors.New(helper.ChannelNotExist)
		}

		pLog.Merchant = "线下USDT"
		pLog.Channel = ch["name"]
	*/
	usdt_info_temp, err := UsdtInfo()
	if err != nil {
		return "", err
	}

	usdt_rate, err := decimal.NewFromString(usdt_info_temp["usdt_rate"])
	if err != nil {
		return "", errors.New(helper.AmountErr)
	}

	dm, err := decimal.NewFromString(amount)
	if err != nil {
		return "", errors.New(helper.AmountErr)
	}

	// 发起的usdt金额
	usdtAmount := dm.Mul(decimal.NewFromInt(1000)).DivRound(usdt_rate, 3).String()

	// 生成我方存款订单号
	orderID := helper.GenId()

	// 检查用户的存款行为是否过于频繁
	err = cacheDepositProcessing(user.UID, time.Now().Unix())
	if err != nil {
		return "", err
	}

	d := g.Record{
		"id":                orderID,
		"prefix":            meta.Prefix,
		"oid":               orderID,
		"uid":               user.UID,
		"top_uid":           user.TopUID,
		"top_name":          user.TopName,
		"parent_name":       user.ParentName,
		"parent_uid":        user.ParentUID,
		"username":          user.Username,
		"channel_id":        p.ChannelID,
		"cid":               p.CateID,
		"pid":               p.ID,
		"amount":            0,
		"state":             DepositConfirming,
		"finance_type":      TransactionUSDTOfflineDeposit,
		"automatic":         "0",
		"created_at":        fctx.Time().In(loc).Unix(),
		"created_uid":       "0",
		"created_name":      "",
		"confirm_at":        "0",
		"confirm_uid":       "0",
		"confirm_name":      "",
		"review_remark":     "",
		"protocol_type":     protocolType,
		"address":           addr,
		"usdt_apply_amount": usdtAmount,
		"rate":              usdt_rate.String(),
		"hash_id":           hashID,
		"flag":              DepositFlagUSDT,
		"level":             user.Level,
	}

	// 请求成功插入订单
	err = deposit(d)
	if err != nil {
		fmt.Println("UsdtPay deposit insert into table error: ", err.Error())
		return "", errors.New(helper.DBErr)
	}

	// 记录存款行为
	_ = cacheDepositProcessingInsert(user.UID, orderID, fctx.Time().In(loc).Unix())

	return orderID, nil
}
