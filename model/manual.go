package model

import (
	"errors"
	"finance/contrib/helper"
	"finance/contrib/validator"
	"fmt"
	"github.com/go-redis/redis/v8"
	"time"

	g "github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/shopspring/decimal"
	"github.com/valyala/fasthttp"
)

// Manual 调用与pid对应的渠道, 发起充值(代付)请求
func ManualPay(fctx *fasthttp.RequestCtx, paymentID, amount string) (string, error) {

	res := map[string]string{}
	user, err := MemberCache(fctx)
	if err != nil {
		return "", err
	}
	fmt.Println("user.Tester", user.Tester)
	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	key := fmt.Sprintf("%s:finance:manual:%s", meta.Prefix, user.Username)
	existCmd := pipe.Exists(ctx, key)
	cmd := pipe.Get(ctx, key)

	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return "", pushLog(err, helper.RedisErr)
	}

	if existCmd.Val() == 1 {
		s, err := cmd.Result()
		if err != nil {
			return "", pushLog(err, helper.RedisErr)
		}

		return s, nil
	}

	ts := fctx.Time().In(loc).Unix()
	p, err := CachePayment(paymentID)
	if err != nil {
		return "", errors.New(helper.ChannelNotExist)
	}

	// 检查存款金额是否符合范围
	a, ok := validator.CheckFloatScope(amount, p.Fmin, p.Fmax)
	if !ok {
		return "", errors.New(helper.AmountOutRange)
	}

	/*
		// 检查用户的存款行为是否过于频繁
		err = cacheDepositProcessing(user.UID, ts)
		if err != nil {
			return res, err
		}
	*/
	amount = a.Truncate(0).String()
	bc, err := BankCardBackend()
	if err != nil {
		fmt.Println("BankCardBackend err = ", err.Error())
		return "", errors.New(helper.BankCardNotExist)
	}

	// 获取附言码
	code, err := TransacCodeGet()
	if err != nil {
		return "", errors.New(helper.ChannelBusyTryOthers)
	}

	fmt.Println("TransacCodeGet code = ", code)

	// 生成我方存款订单号
	orderId := helper.GenId()
	d := g.Record{
		"id":            orderId,
		"prefix":        meta.Prefix,
		"oid":           orderId,
		"uid":           user.UID,
		"top_uid":       user.TopUid,
		"top_name":      user.TopName,
		"parent_name":   user.ParentName,
		"parent_uid":    user.ParentUid,
		"username":      user.Username,
		"channel_id":    p.ChannelID,
		"cid":           p.CateID,
		"pid":           p.ID,
		"amount":        amount,
		"state":         DepositConfirming,
		"finance_type":  helper.TransactionOfflineDeposit,
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
		"tester":  user.Tester,
	}

	// 请求成功插入订单
	err = deposit(d)
	if err != nil {
		fmt.Println("Manual deposit err = ", err)
		return "", pushLog(err, helper.DBErr)
	}

	// 记录存款行为
	_ = cacheDepositProcessingInsert(user.UID, orderId, ts)

	res = map[string]string{
		"id":           orderId,
		"name":         bc.BanklcardName,
		"cardNo":       bc.BanklcardNo,
		"realname":     bc.AccountName,
		"bankAddr":     bc.BankcardAddr,
		"manualRemark": code,
		"ts":           fmt.Sprintf("%d", ts),
	}

	bytes, _ := helper.JsonMarshal(res)
	if user.Tester == "1" {
		_, err = meta.MerchantRedis.Set(ctx, key, string(bytes), 30*time.Minute).Result()
		if err != nil {
			return "", pushLog(err, helper.RedisErr)
		}
	}

	_ = PushWithdrawNotify(depositReviewFmt, user.Username, amount)

	if user.Tester == "0" {
		DepositUpPointReview(orderId, user.UID, "系统", "自动", DepositSuccess)
		CacheDepositProcessingRem(user.UID)
	}
	return string(bytes), nil
}

// DepositManualList 线下转卡订单列表
func ManualList(ex g.Ex, startTime, endTime string, page, pageSize int) (FDepositData, error) {

	ex["prefix"] = meta.Prefix

	data := FDepositData{}

	if startTime != "" && endTime != "" {

		startAt, err := helper.TimeToLoc(startTime, loc)
		if err != nil {
			return data, errors.New(helper.DateTimeErr)
		}

		endAt, err := helper.TimeToLoc(endTime, loc)
		if err != nil {
			return data, errors.New(helper.DateTimeErr)
		}

		ex["created_at"] = g.Op{"between": exp.NewRangeVal(startAt, endAt)}
	}

	if page == 1 {

		total := depositTotal{}
		countQuery, _, _ := dialect.From("tbl_deposit").Select(g.COUNT(1).As("t"), g.SUM("amount").As("s")).Where(ex).ToSQL()
		err := meta.MerchantDB.Get(&total, countQuery)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}

		if total.T.Int64 < 1 {
			return data, nil
		}

		data.Agg = map[string]string{
			"amount": fmt.Sprintf("%.4f", total.S.Float64),
		}
		data.T = total.T.Int64
	}

	offset := uint((page - 1) * pageSize)
	query, _, _ := dialect.From("tbl_deposit").Select(colsDeposit...).
		Where(ex).Offset(offset).Limit(uint(pageSize)).Order(g.C("created_at").Desc()).ToSQL()
	err := meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}

// DepositManualReview 线下转卡-存款审核
func ManualReview(did, remark, name, uid string, state int, record Deposit) error {

	// 加锁
	err := depositLock(did)
	if err != nil {
		return err
	}
	defer depositUnLock(did)

	err = DepositUpPointReview(did, uid, name, remark, state)
	if err == nil {

		if state == DepositSuccess {
			// 清除未未成功的订单计数
			CacheDepositProcessingRem(record.UID)
			amount := decimal.NewFromFloat(record.Amount)

			vals := g.Record{
				"total_finish_amount": g.L(fmt.Sprintf("total_finish_amount+%s", amount.String())),
				"daily_finish_amount": g.L(fmt.Sprintf("daily_finish_amount+%s", amount.String())),
			}
			err = BankCardUpdate(record.BankcardID, vals)
			if err != nil {
				fmt.Println("ManualReview BankCardUpdate = ", err)
				return err
			}

			bc, err := BankCardByID(record.BankcardID)
			if err != nil {
				return err
			}

			totalFinishAmount, _ := decimal.NewFromString(bc.TotalFinishAmount)
			dailyFinishAmount, _ := decimal.NewFromString(bc.DailyFinishAmount)
			totalMaxAmount, _ := decimal.NewFromString(bc.TotalMaxAmount)
			dailyMaxAmount, _ := decimal.NewFromString(bc.DailyMaxAmount)

			if totalFinishAmount.Cmp(totalMaxAmount) >= 0 {

				vals = g.Record{
					"state": "0",
				}
				BankCardUpdate(record.BankcardID, vals)
			}

			if dailyFinishAmount.Cmp(dailyMaxAmount) >= 0 {

				vals = g.Record{
					"state": "0",
				}
				BankCardUpdate(record.BankcardID, vals)
			}
		}

		key := fmt.Sprintf("%s:finance:manual:%s", meta.Prefix, record.Username)
		err = meta.MerchantRedis.Unlink(ctx, key).Err()
		if err != nil {
			_ = pushLog(err, helper.RedisErr)
		}
		return nil
	}

	return err
}
