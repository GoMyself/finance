package model

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/lucacasonato/mqtt"

	"finance/contrib/helper"

	g "github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/shopspring/decimal"
)

// Deposit 存款
type Deposit struct {
	ID              string  `db:"id" json:"id" redis:"id"`                                              //
	Prefix          string  `db:"prefix" json:"prefix" redis:"prefix"`                                  //转账后的金额
	OID             string  `db:"oid" json:"oid" redis:"oid"`                                           //转账前的金额
	UID             string  `db:"uid" json:"uid" redis:"uid"`                                           //用户ID
	Username        string  `db:"username" json:"username" redis:"username"`                            //用户名
	ChannelID       string  `db:"channel_id" json:"channel_id" redis:"channel_id"`                      //
	CID             string  `db:"cid" json:"cid" redis:"cid"`                                           //分类ID
	PID             string  `db:"pid" json:"pid" redis:"pid"`                                           //用户ID
	FinanceType     int     `db:"finance_type" json:"finance_type" redis:"finance_type"`                // 财务类型 441=充值 443=代客充值 445=代理充值
	Amount          float64 `db:"amount" json:"amount" redis:"amount"`                                  //金额
	USDTFinalAmount float64 `db:"usdt_final_amount" json:"usdt_final_amount" redis:"usdt_final_amount"` // 到账金额
	USDTApplyAmount float64 `db:"usdt_apply_amount" json:"usdt_apply_amount" redis:"usdt_apply_amount"` // 提单金额
	Rate            float64 `db:"rate" json:"rate" redis:"rate"`                                        // 汇率
	State           int     `db:"state" json:"state" redis:"state"`                                     //0:待确认:1存款成功2:已取消
	Automatic       int     `db:"automatic" json:"automatic" redis:"automatic"`                         //1:自动转账2:脚本确认3:人工确认
	CreatedAt       int64   `db:"created_at" json:"created_at" redis:"created_at"`                      //
	CreatedUID      string  `db:"created_uid" json:"created_uid" redis:"created_uid"`                   //创建人的ID
	CreatedName     string  `db:"created_name" json:"created_name" redis:"created_name"`                //创建人的名字
	ReviewRemark    string  `db:"review_remark" json:"review_remark" redis:"review_remark"`             //审核备注
	ConfirmAt       int64   `db:"confirm_at" json:"confirm_at" redis:"confirm_at"`                      //确认时间
	ConfirmUID      string  `db:"confirm_uid" json:"confirm_uid" redis:"confirm_uid"`                   //手动确认人id
	ConfirmName     string  `db:"confirm_name" json:"confirm_name" redis:"confirm_name"`                //手动确认人名字
	ProtocolType    string  `db:"protocol_type" json:"protocol_type" redis:"protocol_type"`             //地址类型 trc20 erc20
	Address         string  `db:"address" json:"address" redis:"address"`                               //收款地址
	HashID          string  `db:"hash_id" json:"hash_id" redis:"hash_id"`                               //区块链订单号
	Flag            int     `db:"flag" json:"flag" redis:"flag"`                                        // 1 三方订单 2 三方usdt订单 3 线下转卡订单 4 线下转usdt订单
	BankcardID      string  `db:"bankcard_id" json:"bankcard_id" redis:"bankcard_id"`                   // 线下转卡 收款银行卡id
	ManualRemark    string  `db:"manual_remark" json:"manual_remark" redis:"manual_remark"`             // 线下转卡订单附言
	BankCode        string  `db:"bank_code" json:"bank_code" redis:"bank_code"`                         // 银行编号
	BankNo          string  `db:"bank_no" json:"bank_no" redis:"bank_no"`                               // 银行卡号
	ParentUID       string  `db:"parent_uid" json:"parent_uid" redis:"parent_uid"`                      // 上级uid
	ParentName      string  `db:"parent_name" json:"parent_name" redis:"parent_name"`                   //上级代理名
	TopUID          string  `db:"top_uid" json:"top_uid" redis:"top_uid"`                               // 总代uid
	TopName         string  `db:"top_name" json:"top_name" redis:"top_name"`                            // 总代用户名
	Level           int     `db:"level" json:"level" redis:"level"`
}

// 存款数据
type FDepositData struct {
	T   int64             `json:"t"`
	D   []Deposit         `json:"d"`
	Agg map[string]string `json:"agg"`
}

type depositTotal struct {
	T sql.NullInt64   `json:"t"`
	S sql.NullFloat64 `json:"s"`
}

type DepositData struct {
	FDepositData
	Lock map[string]bool `json:"lock"`
}

// DepositHistory 存款历史列表
func DepositHistory(username, id, channelID, oid, state,
	minAmount, maxAmount, startTime, endTime, cid string, timeFlag uint8, flag, page, pageSize, ty int) (FDepositData, error) {

	data := FDepositData{}
	param := map[string]interface{}{}
	rangeParam := map[string][]interface{}{}

	if username != "" {
		param["username"] = username
	}

	if id != "" {
		param["_id"] = id
	}

	if channelID != "" {
		param["channel_id"] = channelID
	}

	if oid != "" {
		param["oid"] = oid
	}

	if cid != "" {
		param["cid"] = cid
	}

	if state != "" && state != "0" {
		param["state"] = state
	} else {
		rangeParam["state"] = []interface{}{DepositSuccess, DepositCancelled}
	}

	if ty != 0 {
		param["flag"] = ty
	}

	rangeParam["amount"] = []interface{}{0.00, 99999999999.00}
	// 下分列表
	if flag == 1 {
		rangeParam["amount"] = []interface{}{-99999999999.00, 0.00}
	}

	if minAmount != "" && maxAmount != "" {
		minF, err := strconv.ParseFloat(minAmount, 64)
		if err != nil {
			return data, pushLog(err, helper.AmountErr)
		}

		maxF, err := strconv.ParseFloat(maxAmount, 64)
		if err != nil {
			return data, pushLog(err, helper.AmountErr)
		}

		rangeParam["amount"] = []interface{}{minF, maxF}
	}

	if startTime != "" && endTime != "" {

		startAt, err := helper.TimeToLoc(startTime, loc)
		if err != nil {
			return data, errors.New(helper.DateTimeErr)
		}

		endAt, err := helper.TimeToLoc(endTime, loc)
		if err != nil {
			return data, errors.New(helper.DateTimeErr)
		}

		if timeFlag == 1 {
			rangeParam["created_at"] = []interface{}{startAt, endAt}
		} else {
			rangeParam["confirm_at"] = []interface{}{startAt, endAt}
		}
	}

	aggField := map[string]string{"amount_agg": "amount"}
	return DepositESQuery(esPrefixIndex("tbl_deposit"), "created_at", page, pageSize, param, rangeParam, aggField)
}

func DepositDetail(username, state, channelID, timeFlag, startTime, endTime string, page, pageSize int) (FDepositData, error) {

	data := FDepositData{}
	ex := g.Ex{
		"amount": g.Op{"gt": 0.00},
		"prefix": meta.Prefix,
	}

	if username != "" {
		ex["username"] = username
	}

	if channelID != "" {
		ex["channel_id"] = channelID
	}

	if state != "" {
		ex["state"] = state
	}

	order := "created_at"
	if startTime != "" && endTime != "" {

		startAt, err := helper.TimeToLoc(startTime, loc)
		if err != nil {
			return data, errors.New(helper.DateTimeErr)
		}

		endAt, err := helper.TimeToLoc(endTime, loc)
		if err != nil {
			return data, errors.New(helper.DateTimeErr)
		}

		if timeFlag == "2" {
			order = "confirm_at"
			ex["confirm_at"] = g.Op{"between": exp.NewRangeVal(startAt, endAt)}
		} else {
			ex["created_at"] = g.Op{"between": exp.NewRangeVal(startAt, endAt)}
		}
	}

	if page == 1 {

		var total depositTotal
		query, _, _ := dialect.From("tbl_deposit").Select(g.COUNT(1).As("t"), g.SUM("amount").As("s")).Where(ex).ToSQL()
		err := meta.MerchantDB.Get(&total, query)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}

		if total.T.Int64 < 1 {
			return data, nil
		}

		data.Agg = map[string]string{
			"amount": fmt.Sprintf("%.4f", total.S.Float64),
		}

		// 查询到账金额和上分金额 (当前需求到账金额和上分金额用一个字段)
		exc := g.Ex{
			"prefix": meta.Prefix,
		}
		for k, v := range ex {
			exc[k] = v
		}
		exc["state"] = DepositSuccess
		query, _, _ = dialect.From("tbl_deposit").Select(g.COALESCE(g.SUM("amount"), 0).As("s")).Where(exc).ToSQL()
		err = meta.MerchantDB.Get(&total, query)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}

		data.Agg["valid_amount"] = fmt.Sprintf("%.4f", total.S.Float64)
		data.T = total.T.Int64
	}

	offset := uint((page - 1) * pageSize)
	query, _, _ := dialect.From("tbl_deposit").Select(colsDeposit...).
		Where(ex).Offset(offset).Limit(uint(pageSize)).Order(g.C(order).Desc()).ToSQL()
	err := meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}

// DepositList 存款订单列表
func DepositList(ex g.Ex, startTime, endTime string, page, pageSize int) (DepositData, error) {

	ex["prefix"] = meta.Prefix
	data := DepositData{}

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

	var ids []string
	for _, v := range data.D {
		ids = append(ids, v.UID)
	}
	data.Lock, _ = lockMapByUids(ids)

	return data, nil
}

// DepositReview 存款补单审核
func DepositReview(did, remark, state, name, uid string) error {

	// 加锁
	err := depositLock(did)
	if err != nil {
		return err
	}
	defer depositUnLock(did)

	iState, _ := strconv.Atoi(state)

	ex := g.Ex{"id": did, "state": DepositConfirming}
	order, err := DepositOrderFindOne(ex)
	if err != nil {
		return err
	}

	// 充值成功处理订单状态
	if iState == DepositSuccess {
		_ = CacheDepositProcessingRem(order.UID)
	}

	err = DepositUpPoint(did, uid, name, remark, iState)
	if err != nil {
		return err
	}

	return nil
}

//存款上分
func DepositUpPoint(did, uid, name, remark string, state int) error {

	// 判断状态是否合法
	allow := map[int]bool{
		DepositCancelled: true,
		DepositSuccess:   true,
	}
	if _, ok := allow[state]; !ok {
		return errors.New(helper.OrderStateErr)
	}

	// 判断订单是否存在
	ex := g.Ex{"id": did, "state": DepositConfirming}
	order, err := DepositOrderFindOne(ex)
	if err != nil {
		return err
	}

	// 如果已经有一笔订单补单成功,则其他订单不允许补单成功
	if DepositSuccess == state {
		// 这里的ex不能覆盖上面的ex
		_, err = DepositOrderFindOne(g.Ex{"oid": order.OID, "state": DepositSuccess})
		if err != nil && err.Error() != helper.OrderNotExist {
			return err
		}

		if err == nil {
			return errors.New(helper.OrderExist)
		}
	}

	now := time.Now()
	record := g.Record{
		"state":         state,
		"confirm_at":    now.Unix(),
		"confirm_uid":   uid,
		"confirm_name":  name,
		"review_remark": remark,
	}
	query, _, _ := dialect.Update("tbl_deposit").Set(record).Where(ex).ToSQL()
	fmt.Println(query)
	money := decimal.NewFromFloat(order.Amount)
	amount := money.String()
	cashType := helper.TransactionDeposit
	if money.Cmp(zero) == -1 {
		cashType = helper.TransactionFinanceDownPoint
		amount = money.Abs().String()
	}

	switch state {
	case DepositCancelled:
		// 存款失败 直接修改订单状态
		if cashType == helper.TransactionDeposit {
			_, err = meta.MerchantDB.Exec(query)
			if err != nil {
				return pushLog(err, helper.DBErr)
			}
			//发送推送
			msg := fmt.Sprintf(`{"ty":"1","amount": "%f", "ts":"%d","status":"faild"}`, order.Amount, time.Now().Unix())
			fmt.Println(msg)
			topic := fmt.Sprintf("%s/%s/finance", meta.Prefix, order.UID)
			err = meta.MerchantMqtt.Publish(ctx, topic, []byte(msg), mqtt.AtLeastOnce)
			if err != nil {
				fmt.Println("merchantNats.Publish finance = ", err.Error())
				return err
			}
			return nil
		}
		// 存款成功 和 下分失败switch完成后处理
	case DepositSuccess:
		// 下分成功 修改订单状态并修改adjust表的审核状态
		if cashType == helper.TransactionFinanceDownPoint {
			//开启事务
			tx, err := meta.MerchantDB.Begin()
			if err != nil {
				return pushLog(err, helper.DBErr)
			}

			_, err = tx.Exec(query)
			if err != nil {
				_ = tx.Rollback()
				return pushLog(err, helper.DBErr)
			}

			r := g.Record{
				"state":          AdjustReviewPass,
				"review_at":      now.Unix(),
				"review_uid":     uid,
				"review_name":    name,
				"review_remark":  remark,
				"hand_out_state": AdjustSuccess,
			}
			query, _, _ = dialect.Update("tbl_member_adjust").Set(r).Where(g.Ex{"id": order.OID}).ToSQL()
			fmt.Println(query)
			_, err = tx.Exec(query)
			if err != nil {
				_ = tx.Rollback()
				return pushLog(err, helper.DBErr)
			}

			err = tx.Commit()
			if err != nil {
				return pushLog(err, helper.DBErr)
			}

			return nil
		}
		// 存款成功 和 下分失败switch完成后处理
	default:
		// 代码不会走到这边来, 此行的目的只是为了switch case 的完整性
		return errors.New(helper.OrderStateErr)
	}

	// 后面都是存款成功 和 下分失败 的处理
	// 1、查询用户额度
	balance, err := GetBalanceDB(order.UID)
	if err != nil {
		return err
	}
	balanceAfter := decimal.NewFromFloat(balance.Balance).Add(money)

	// 开启事务
	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	// 2、更新订单状态
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	// 如果是下分 审核失败
	if DepositCancelled == state && cashType == helper.TransactionFinanceDownPoint {
		// 修改状态
		r := g.Record{
			"state":          AdjustReviewReject,
			"review_at":      now.Unix(),
			"review_uid":     uid,
			"review_name":    name,
			"review_remark":  remark,
			"hand_out_state": AdjustFailed,
		}
		query, _, _ = dialect.Update("tbl_member_adjust").Set(r).Where(g.Ex{"id": order.OID}).ToSQL()
		_, err = tx.Exec(query)
		if err != nil {
			_ = tx.Rollback()
			return pushLog(err, helper.DBErr)
		}

		balanceAfter = decimal.NewFromFloat(balance.Balance).Add(money.Abs())
	}

	balanceFeeAfter := balanceAfter
	fee := decimal.Zero
	var feeCashType int
	//如果存款有优惠
	key := meta.Prefix + ":p:c:t:" + order.ChannelID
	promoState, err := meta.MerchantRedis.HGet(ctx, key, "promo_state").Result()
	if err != nil && err != redis.Nil {
		//缓存没有配置就跳过
		fmt.Println(err)
	}
	//开启了优惠
	if promoState == "1" {
		promoDiscount, err := meta.MerchantRedis.HGet(ctx, key, "promo_discount").Result()
		if err != nil && err != redis.Nil {
			//缓存没有配置就跳过
			fmt.Println(err)
		}
		pd, _ := decimal.NewFromString(promoDiscount)
		fmt.Println("promoDiscount:", promoDiscount)
		if pd.GreaterThan(decimal.Zero) {
			//大于0就是优惠，给钱
			fee = money.Mul(pd).Div(decimal.NewFromInt(100))
			money = money.Add(fee)
			balanceFeeAfter = decimal.NewFromFloat(balance.Balance).Add(money.Abs())
			feeCashType = helper.TransactionDepositBonus
		} else if pd.LessThan(decimal.Zero) {
			//小于0就是收费，扣钱
			fee = money.Mul(pd).Div(decimal.NewFromInt(100))
			money = money.Sub(fee)
			balanceFeeAfter = decimal.NewFromFloat(balance.Balance).Add(money.Abs())
			feeCashType = helper.TransactionDepositFee

		}
	}

	// 3、更新余额
	ex = g.Ex{
		"uid":    order.UID,
		"prefix": meta.Prefix,
	}
	br := g.Record{
		"balance": g.L(fmt.Sprintf("balance+%s", amount)),
	}
	query, _, _ = dialect.Update("tbl_members").Set(br).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	// 4、新增账变记录
	id := helper.GenId()
	mbTrans := memberTransaction{
		AfterAmount:  balanceAfter.String(),
		Amount:       amount,
		BeforeAmount: decimal.NewFromFloat(balance.Balance).String(),
		BillNo:       order.ID,
		CreatedAt:    now.UnixMilli(),
		ID:           id,
		CashType:     cashType,
		UID:          order.UID,
		Username:     order.Username,
		Prefix:       meta.Prefix,
	}
	if cashType == helper.TransactionFinanceDownPoint {
		mbTrans.OperationNo = id
	}

	query, _, _ = dialect.Insert("tbl_balance_transaction").Rows(mbTrans).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	if balanceFeeAfter.Cmp(balanceAfter) != 0 {
		//手续费/优惠的帐变
		id = helper.GenId()
		mbTrans = memberTransaction{
			AfterAmount:  balanceFeeAfter.String(),
			Amount:       fee.String(),
			BeforeAmount: balanceAfter.String(),
			BillNo:       order.ID,
			CreatedAt:    time.Now().UnixMilli(),
			ID:           id,
			CashType:     feeCashType,
			UID:          order.UID,
			Username:     order.Username,
			Prefix:       meta.Prefix,
		}

		query, _, _ = dialect.Insert("tbl_balance_transaction").Rows(mbTrans).ToSQL()
		_, err = tx.Exec(query)
		if err != nil {
			_ = tx.Rollback()
			return pushLog(err, helper.DBErr)
		}
	}

	err = tx.Commit()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	fmt.Println("state:", state)
	if DepositSuccess == state {

		rec := g.Record{
			"first_deposit_at":     order.CreatedAt,
			"first_deposit_amount": order.Amount,
		}
		ex1 := g.Ex{
			"username":         order.Username,
			"first_deposit_at": 0,
		}
		query, _, _ = dialect.Update("tbl_members").Set(rec).Where(ex1).ToSQL()
		fmt.Printf("memberFirstDeposit Update: %v\n", query)
		_, err := meta.MerchantDB.Exec(query)
		if err != nil {
			fmt.Println("update member first_amount err:", err.Error())
		}

		rec2 := g.Record{
			"second_deposit_at":     order.CreatedAt,
			"second_deposit_amount": order.Amount,
		}
		ex2 := g.Ex{
			"username":          order.Username,
			"second_deposit_at": 0,
		}
		query, _, _ = dialect.Update("tbl_members").Set(rec2).Where(ex2).ToSQL()
		fmt.Printf("memberSecondDeposit Update: %v\n", query)
		_, err = meta.MerchantDB.Exec(query)
		if err != nil {
			fmt.Println("update member second_amount err:", err.Error())
		}

		//发送站内信
		title := "Thông Báo Nạp Tiền Thành Công"
		content := fmt.Sprintf("Quý Khách Của P3 Thân Mến:\nBạn Đã Nạp Tiền Thành Công %s KVND,Vui Lòng KIểm Tra Ngay,Nếu Bạn Có Bất Cứ Thắc Mắc Vấn Đề Gì Vui Lòng Liên Hệ CSKH Để Biết Thêm Chi Tiết.【P3】Chúc Bạn Cược Đâu Thắng Đó !!\n",
			decimal.NewFromFloat(order.Amount).Truncate(0).String())
		err = messageSend(order.ID, title, "", content, "system", meta.Prefix, 0, 0, 1, []string{order.Username})
		if err != nil {
			_ = pushLog(err, helper.ESErr)
		}
		//发送推送
		msg := fmt.Sprintf(`{"ty":"1","amount": "%f", "ts":"%d","status":"success"}`, order.Amount, time.Now().Unix())
		fmt.Println(msg)
		topic := fmt.Sprintf("%s/%s/finance", meta.Prefix, order.UID)
		err = meta.MerchantMqtt.Publish(ctx, topic, []byte(msg), mqtt.AtLeastOnce)
		if err != nil {
			fmt.Println("merchantNats.Publish finance = ", err.Error())
			return err
		}

	} else {
		//发送推送
		msg := fmt.Sprintf(`{"ty":"1","amount": "%f", "ts":"%d","status":"faild"}`, order.Amount, time.Now().Unix())
		fmt.Println(msg)
		topic := fmt.Sprintf("%s/%s/finance", meta.Prefix, order.UID)
		err = meta.MerchantMqtt.Publish(ctx, topic, []byte(msg), mqtt.AtLeastOnce)
		if err != nil {
			fmt.Println("merchantNats.Publish finance = ", err.Error())
			return err
		}
	}

	_ = MemberUpdateCache(order.Username)
	return nil
}

// DepositOrderFindOne 查询存款订单
func DepositOrderFindOne(ex g.Ex) (Deposit, error) {

	ex["prefix"] = meta.Prefix
	order := Deposit{}
	query, _, _ := dialect.From("tbl_deposit").Select(colsDeposit...).Where(ex).Limit(1).ToSQL()
	fmt.Println(query)
	err := meta.MerchantDB.Get(&order, query)
	if err == sql.ErrNoRows {
		return order, errors.New(helper.OrderNotExist)
	}

	if err != nil {
		return order, pushLog(err, helper.DBErr)
	}

	return order, nil
}

// DepositManual 手动补单
func DepositManual(id, amount, remark, name, uid string) error {

	money, _ := decimal.NewFromString(amount)
	if money.Cmp(zero) < 1 {
		return errors.New(helper.AmountErr)
	}
	// 判断订单是否存在
	oEx := g.Ex{"id": id, "automatic": 1}
	order, err := DepositOrderFindOne(oEx)
	if err != nil {
		return err
	}

	err = lockMemberCheck(order.Username)
	if err != nil {
		return err
	}

	// 判断状态
	if order.State != DepositConfirming {
		return errors.New(helper.OrderStateErr)
	}

	// 判断此订单是否已经已经有一笔补单成功,如果这笔订单的手动补单有一笔成功,则不允许再补单
	existEx := g.Ex{
		"oid":       order.OID,
		"state":     DepositSuccess,
		"automatic": 0,
		"prefix":    meta.Prefix,
	}
	_, err = DepositOrderFindOne(existEx)
	if err != nil && err.Error() == helper.DBErr {
		return err
	}
	if err == nil {
		return errors.New(helper.OrderExist)
	}

	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return errors.New(helper.TransErr)
	}

	now := time.Now()
	// 生成订单
	d := g.Record{
		"id":                helper.GenId(),
		"prefix":            meta.Prefix,
		"oid":               order.OID,
		"uid":               order.UID,
		"username":          order.Username,
		"channel_id":        order.ChannelID,
		"cid":               order.CID,
		"pid":               order.PID,
		"amount":            amount,
		"state":             DepositConfirming,
		"automatic":         "0",
		"created_at":        fmt.Sprintf("%d", now.Unix()),
		"created_uid":       uid,
		"created_name":      name,
		"confirm_at":        "0",
		"confirm_uid":       "0",
		"confirm_name":      "",
		"review_remark":     remark,
		"finance_type":      helper.TransactionDeposit,
		"top_uid":           order.TopUID,
		"top_name":          order.TopName,
		"parent_uid":        order.ParentUID,
		"parent_name":       order.ParentName,
		"manual_remark":     order.ManualRemark,
		"bankcard_id":       order.BankcardID,
		"protocol_type":     order.ProtocolType,
		"rate":              order.Rate,
		"usdt_final_amount": order.USDTFinalAmount,
		"usdt_apply_amount": order.USDTApplyAmount,
		"address":           order.Address,
		"hash_id":           order.HashID,
		"flag":              order.Flag,
		"bank_code":         order.BankCode,
		"bank_no":           order.BankNo,
		"level":             order.Level,
	}
	query, _, _ := dialect.Insert("tbl_deposit").Rows(d).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		fmt.Println("deposit err = ", err)
		_ = tx.Rollback()
		return errors.New(helper.TransErr)
	}

	// 判断状态，如果处理中则更新状态
	if order.State == DepositConfirming {
		// 更新配置
		ex := g.Ex{"id": order.ID, "prefix": meta.Prefix, "state": DepositConfirming}
		recs := g.Record{
			"state":         DepositCancelled,
			"confirm_at":    now.Unix(),
			"automatic":     "0",
			"confirm_uid":   uid,
			"confirm_name":  name,
			"review_remark": remark,
		}
		query, _, _ = dialect.Update("tbl_deposit").Set(recs).Where(ex).ToSQL()
		r, err := tx.Exec(query)
		fmt.Println(r)
		fmt.Println(err)
		if err != nil {
			_ = tx.Rollback()
			return errors.New(helper.TransErr)
		}
		refectRows, err := r.RowsAffected()
		if err != nil {
			_ = tx.Rollback()
			return errors.New(helper.TransErr)
		}
		fmt.Println(refectRows)

		if refectRows == 0 {
			_ = tx.Rollback()
			return errors.New(helper.TransErr)
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.New(helper.TransErr)
	}

	// 发送消息通知
	_ = PushMerchantNotify(manualReviewFmt, name, order.Username, amount)

	return nil
}

// DepositReduce 存款下分
func DepositReduce(username, amount, remark, name, uid string) error {

	//下分的额度是负数
	money, _ := decimal.NewFromString(amount)
	if money.Cmp(zero) < 1 {
		return errors.New(helper.AmountErr)
	}

	//查询用户名
	mb, err := MemberGetByName(username)
	if err != nil || len(mb.Username) < 1 {
		return errors.New(helper.UserNotExist)
	}

	//查询用户额度
	balance, err := GetBalanceDB(mb.UID)
	if err != nil {
		return err
	}

	if money.Cmp(decimal.NewFromFloat(balance.Balance)) == 1 {
		return errors.New(helper.LackOfBalance)
	}
	balanceAfter := decimal.NewFromFloat(balance.Balance).Sub(money)

	//开启事务
	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	//扣除余额
	ex := g.Ex{
		"uid":    mb.UID,
		"prefix": meta.Prefix,
	}
	br := g.Record{
		"balance": g.L(fmt.Sprintf("balance-%s", money.String())),
	}
	query, _, _ := dialect.Update("tbl_members").Set(br).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	now := time.Now()
	//生成订单
	id := helper.GenId()
	d := g.Record{
		"id":            id,
		"prefix":        meta.Prefix,
		"oid":           id,
		"uid":           mb.UID,
		"parent_uid":    mb.ParentUid,
		"parent_name":   mb.ParentName,
		"top_uid":       mb.TopUid,
		"top_name":      mb.TopName,
		"username":      mb.Username,
		"cid":           "0",
		"pid":           "0",
		"channel_id":    "0",
		"amount":        fmt.Sprintf("-%s", amount),
		"state":         DepositConfirming,
		"automatic":     "0",
		"created_at":    fmt.Sprintf("%d", now.In(loc).Unix()),
		"created_uid":   uid,
		"created_name":  name,
		"confirm_at":    "0",
		"confirm_uid":   "0",
		"confirm_name":  "",
		"review_remark": remark,
		"finance_type":  helper.TransactionDeposit,
		"level":         mb.Level,
	}
	query, _, _ = dialect.Insert("tbl_deposit").Rows(d).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	//7、新增账变记录
	mbTrans := memberTransaction{
		AfterAmount:  balanceAfter.String(),
		Amount:       amount,
		BeforeAmount: decimal.NewFromFloat(balance.Balance).String(),
		BillNo:       id,
		CreatedAt:    now.UnixNano() / 1e6,
		ID:           id,
		CashType:     helper.TransactionFinanceDownPoint,
		UID:          mb.UID,
		Username:     mb.Username,
		Prefix:       meta.Prefix,
	}

	query, _, _ = dialect.Insert("tbl_balance_transaction").Rows(mbTrans).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	// 插入财务调整记录
	record := g.Record{
		"id":             id,
		"uid":            mb.UID,
		"ty":             1, // 后台调整
		"prefix":         meta.Prefix,
		"username":       mb.Username,
		"top_uid":        mb.TopUid,
		"top_name":       mb.TopName,
		"parent_uid":     mb.ParentUid,
		"parent_name":    mb.ParentName,
		"amount":         fmt.Sprintf("-%s", amount),
		"adjust_type":    1,
		"adjust_mode":    AdjustDownMode,
		"is_turnover":    0,
		"turnover_multi": 0,
		"apply_remark":   remark,
		"images":         "",
		"state":          AdjustReviewing, // 状态:256=审核中,257=同意, 258=拒绝
		"apply_at":       uint32(now.Unix()),
		"apply_uid":      uid,  // 申请人
		"apply_name":     name, // 申请人
	}

	query, _, _ = dialect.Insert("tbl_member_adjust").Rows(record).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	MemberUpdateCache(mb.Username)
	// 发送消息通知
	_ = PushMerchantNotify(downgradeReviewFmt, name, username, amount)

	return nil
}

// depositLock 锁定充值订单 防止并发多充钱
func depositLock(id string) error {

	key := fmt.Sprintf(depositOrderLockKey, id)
	return Lock(key)
}

// depositUnLock 解锁充值订单
func depositUnLock(id string) {
	key := fmt.Sprintf(depositOrderLockKey, id)
	Unlock(key)
}

func DepositFindOne(id string) (Deposit, error) {

	ex := g.Ex{
		"id": id,
	}

	return DepositOrderFindOne(ex)
}

// DepositRecordUpdate 更新订单信息
func DepositRecordUpdate(id string, record g.Record) error {

	ex := g.Ex{
		"id": id,
	}
	toSQL, _, _ := dialect.Update("tbl_deposit").Where(ex).Set(record).ToSQL()
	_, err := meta.MerchantDB.Exec(toSQL)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return nil
}

func DepositUpPointReview(did, uid, name, remark string, state int) error {

	// 判断状态是否合法
	allow := map[int]bool{
		DepositCancelled: true,
		DepositSuccess:   true,
	}
	if _, ok := allow[state]; !ok {
		return errors.New(helper.OrderStateErr)
	}

	// 判断订单是否存在
	ex := g.Ex{"id": did, "state": []int{DepositReviewing, DepositConfirming}}
	order, err := DepositOrderFindOne(ex)
	if err != nil {
		return err
	}

	now := time.Now()
	record := g.Record{
		"state":         state,
		"confirm_at":    now.Unix(),
		"confirm_uid":   uid,
		"confirm_name":  name,
		"review_remark": remark,
	}
	query, _, _ := dialect.Update("tbl_deposit").Set(record).Where(ex).ToSQL()

	money := decimal.NewFromFloat(order.Amount)
	amount := money.String()

	if DepositCancelled == state {
		_, err = meta.MerchantDB.Exec(query)
		if err != nil {
			return pushLog(err, helper.DBErr)
		}

		return nil
	}

	// 后面都是存款成功 和 下分失败 的处理
	// 1、查询用户额度
	balance, err := GetBalanceDB(order.UID)
	if err != nil {
		return err
	}
	balanceAfter := decimal.NewFromFloat(balance.Balance).Add(money)

	balanceFeeAfter := balanceAfter
	fee := decimal.Zero
	var feeCashType int
	//如果存款有优惠
	key := meta.Prefix + ":p:c:t:" + order.ChannelID
	promoState, err := meta.MerchantRedis.HGet(ctx, key, "promo_state").Result()
	if err != nil && err != redis.Nil {
		//缓存没有配置就跳过
		fmt.Println(err)
	}
	//开启了优惠
	if promoState == "1" {
		promoDiscount, err := meta.MerchantRedis.HGet(ctx, key, "promo_discount").Result()
		if err != nil && err != redis.Nil {
			//缓存没有配置就跳过
			fmt.Println(err)
		}
		pd, _ := decimal.NewFromString(promoDiscount)
		fmt.Println("promoDiscount:", promoDiscount)
		if pd.GreaterThan(decimal.Zero) {
			//大于0就是优惠，给钱
			fee = money.Mul(pd).Div(decimal.NewFromInt(100))
			money = money.Add(fee)
			balanceFeeAfter = decimal.NewFromFloat(balance.Balance).Add(money.Abs())
			feeCashType = helper.TransactionDepositBonus
		} else if pd.LessThan(decimal.Zero) {
			//小于0就是收费，扣钱
			fee = money.Mul(pd).Div(decimal.NewFromInt(100))
			money = money.Sub(fee)
			balanceFeeAfter = decimal.NewFromFloat(balance.Balance).Add(money.Abs())
			feeCashType = helper.TransactionDepositFee

		}
	}

	// 开启事务
	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	// 2、更新订单状态
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	// 3、更新余额
	ex = g.Ex{
		"uid":    order.UID,
		"prefix": meta.Prefix,
	}
	br := g.Record{
		"balance": g.L(fmt.Sprintf("balance+%s", amount)),
	}
	query, _, _ = dialect.Update("tbl_members").Set(br).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	// 4、新增账变记录
	id := helper.GenId()
	mbTrans := memberTransaction{
		AfterAmount:  balanceAfter.String(),
		Amount:       amount,
		BeforeAmount: decimal.NewFromFloat(balance.Balance).String(),
		BillNo:       order.ID,
		CreatedAt:    now.UnixMilli(),
		ID:           id,
		CashType:     helper.TransactionDeposit,
		UID:          order.UID,
		Username:     order.Username,
		Prefix:       meta.Prefix,
	}

	query, _, _ = dialect.Insert("tbl_balance_transaction").Rows(mbTrans).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	if balanceFeeAfter.Cmp(balanceAfter) != 0 {
		//手续费/优惠的帐变
		id = helper.GenId()
		mbTrans = memberTransaction{
			AfterAmount:  balanceFeeAfter.String(),
			Amount:       fee.String(),
			BeforeAmount: balanceAfter.String(),
			BillNo:       order.ID,
			CreatedAt:    time.Now().UnixMilli(),
			ID:           id,
			CashType:     feeCashType,
			UID:          order.UID,
			Username:     order.Username,
			Prefix:       meta.Prefix,
		}

		query, _, _ = dialect.Insert("tbl_balance_transaction").Rows(mbTrans).ToSQL()
		_, err = tx.Exec(query)
		if err != nil {
			_ = tx.Rollback()
			return pushLog(err, helper.DBErr)
		}
	}

	err = tx.Commit()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	_ = MemberUpdateCache(order.Username)

	fmt.Println(state)
	if DepositSuccess == state {

		rec := g.Record{
			"first_deposit_at":     order.CreatedAt,
			"first_deposit_amount": order.Amount,
		}
		ex1 := g.Ex{
			"username":         order.Username,
			"first_deposit_at": 0,
		}
		query, _, _ = dialect.Update("tbl_members").Set(rec).Where(ex1).ToSQL()
		fmt.Printf("memberFirstDeposit Update: %v\n", query)
		_, err := meta.MerchantDB.Exec(query)
		if err != nil {
			fmt.Println("update member first_amount err:", err.Error())
		}

		rec2 := g.Record{
			"second_deposit_at":     order.CreatedAt,
			"second_deposit_amount": order.Amount,
		}
		ex2 := g.Ex{
			"username":          order.Username,
			"second_deposit_at": 0,
		}
		query, _, _ = dialect.Update("tbl_members").Set(rec2).Where(ex2).ToSQL()
		fmt.Printf("memberSecondDeposit Update: %v\n", query)
		_, err = meta.MerchantDB.Exec(query)
		if err != nil {
			fmt.Println("update member second_amount err:", err.Error())
		}

		//发送站内信
		title := "Thông Báo Nạp Tiền Thành Công"
		content := fmt.Sprintf("Quý Khách Của P3 Thân Mến:\nBạn Đã Nạp Tiền Thành Công %s KVND,Vui Lòng KIểm Tra Ngay,Nếu Bạn Có Bất Cứ Thắc Mắc Vấn Đề Gì Vui Lòng Liên Hệ CSKH Để Biết Thêm Chi Tiết.【P3】Chúc Bạn Cược Đâu Thắng Đó !!\n",
			decimal.NewFromFloat(order.Amount).Truncate(0).String())
		err = messageSend(order.ID, title, "", content, "system", meta.Prefix, 0, 0, 1, []string{order.Username})
		if err != nil {
			_ = pushLog(err, helper.ESErr)
		}

		//发送推送
		msg := fmt.Sprintf(`{"ty":"1","amount": "%f", "ts":"%d","status":"success"}`, order.Amount, time.Now().Unix())
		fmt.Println(msg)
		topic := fmt.Sprintf("%s/%s/finance", meta.Prefix, order.UID)
		err = meta.MerchantMqtt.Publish(ctx, topic, []byte(msg), mqtt.AtLeastOnce)
		if err != nil {
			fmt.Println("merchantNats.Publish finance = ", err.Error())
			return err
		}
	}
	return nil
}

// DepositUSDTReview 线下USDT-存款审核
func DepositUSDTReview(did, remark, name, adminUID, depositUID string, state int) error {

	// 加锁
	err := depositLock(did)
	if err != nil {
		return err
	}
	defer depositUnLock(did)

	err = DepositUpPointReview(did, adminUID, name, remark, state)
	if err != nil {
		return err
	}

	// 充值成功处理订单状态
	if state == DepositSuccess {
		_ = CacheDepositProcessingRem(depositUID)
	}

	return nil
}

/**
1：如果用户有连续10笔未支付订单，则限制30分钟不能存款
（点击存款页面，确定存款按钮时候，轻提示报错：由于您有大量未支付订单，请于30分钟后再试）

2：30分钟后,用户可再次发起存款，如果又有5笔未支付订单，则限制24小时内不能存款
（点击存款页面，确定存款按钮时候，轻提示报错：由于您有大量未支付订单，请于24小时后再试）

3：24小时后。用户可再次发起存款，如果又有5笔未支付订单，则限制24小时内不能存款
（点击存款页面，确定存款按钮时候，轻提示报错：由于您有大量未支付订单，请于24小时后再试）

注：有任何一笔存款成功的订单后，此限制就会重置列

用户前9笔订单都是未支付，第10笔订单支付成功，，此时用户存款限制就会重置，又可发起10次未支付订单才会被限制
*/

// 限制用户存款频率
func cacheDepositProcessing(uid string, now int64) error {

	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	// 检查是否被手动锁定
	manual_lock_key := fmt.Sprintf("%s:finance:mlock:%s", meta.Prefix, uid)
	automatic_lock_key := fmt.Sprintf("%s:finance:alock:%s", meta.Prefix, uid)

	exists := pipe.Exists(ctx, manual_lock_key)

	// 检查是否被自动锁定
	rs := pipe.ZRevRangeWithScores(ctx, automatic_lock_key, 0, -1)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}

	if exists.Val() != 0 {
		return errors.New(helper.NoChannelErr)
	}

	recs, err := rs.Result()
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}

	num := len(recs)
	if num < 10 {
		return nil
	}

	// 十笔 订单 锁定 5 分钟
	if num == 20 && now < int64(recs[0].Score)+5*60 {
		// 最后一笔订单的时间
		return errors.New(helper.EmptyOrder30MinsBlock)
	}

	// 超出10笔 每隔五笔限制24小时
	if num > 10 && num%5 == 0 && now < int64(recs[0].Score)+24*60*60 {
		return errors.New(helper.EmptyOrder5HoursBlock)
	}

	return nil
}

func cacheDepositProcessingInsert(uid, depositId string, now int64) error {

	automatic_lock_key := fmt.Sprintf("%s:finance:alock:%s", meta.Prefix, uid)

	z := redis.Z{
		Score:  float64(now),
		Member: depositId,
	}
	return meta.MerchantRedis.ZAdd(ctx, automatic_lock_key, &z).Err()
}

// CacheDepositProcessingRem 清除未未成功的订单计数
func CacheDepositProcessingRem(uid string) error {

	automatic_lock_key := fmt.Sprintf("%s:finance:alock:%s", meta.Prefix, uid)
	return meta.MerchantRedis.Unlink(ctx, automatic_lock_key).Err()
}

// 存入数据库
func deposit(record g.Record) error {

	query, _, _ := dialect.Insert("tbl_deposit").Rows(record).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return err

	}

	return nil
}

// 获取订单信息
func depositFind(id string) (Deposit, error) {

	d := Deposit{}

	ex := g.Ex{"id": id}
	query, _, _ := dialect.From("tbl_deposit").Select(colsDeposit...).Where(ex).Limit(1).ToSQL()
	fmt.Println(query)
	err := meta.MerchantDB.Get(&d, query)
	if err == sql.ErrNoRows {
		return d, errors.New(helper.OrderNotExist)
	}

	if err != nil {
		return d, pushLog(err, helper.DBErr)
	}

	return d, nil
}

func depositUpdate(state int, order Deposit) error {

	// 加锁
	err := depositLock(order.ID)
	if err != nil {
		return err
	}
	defer depositUnLock(order.ID)

	// 充值成功处理订单状态
	if state == DepositSuccess {
		_ = CacheDepositProcessingRem(order.UID)
	}

	err = DepositUpPoint(order.ID, "0", "", "", state)
	if err != nil {
		return err
	}

	return nil
}

func depositUpdateUsdtAmount(ID string, usdtAmount, hashID string, rate float64) error {

	usdt, err := decimal.NewFromString(usdtAmount)
	if err != nil {
		return err
	}

	rec := g.Record{
		"usdt_final_amount": usdtAmount,
		"hash_id":           hashID,
		"amount":            usdt.Mul(decimal.NewFromFloat(rate)).DivRound(decimal.NewFromInt(1000), 4).String(),
	}

	ex := g.Ex{
		"id": ID,
	}

	q, _, _ := dialect.Update("tbl_deposit").Set(rec).Where(ex).ToSQL()
	_, err = meta.MerchantDB.Exec(q)
	return err
}

// 获取用户上笔订单存款金额
func depositLastAmount(uid string) (float64, error) {

	ex := g.Ex{
		"uid":    uid,
		"state":  DepositSuccess,
		"amount": g.Op{"gt": 0},
	}
	var amount float64
	query, _, _ := dialect.From("tbl_deposit").Select("amount").Where(ex).Order(g.I("created_at").Desc()).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&amount, query)
	if err != nil && err != sql.ErrNoRows {
		return amount, pushLog(err, helper.DBErr)
	}

	return amount, nil
}
