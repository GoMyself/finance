package model

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"

	"finance/contrib/helper"
	"finance/contrib/validator"

	g "github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/olivere/elastic/v7"
	"github.com/shopspring/decimal"
	"github.com/valyala/fasthttp"
)

// Withdraw 会员提款表
type Withdraw struct {
	ID                string  `db:"id"                  json:"id"                 redis:"id"`
	Prefix            string  `db:"prefix"              json:"prefix"             redis:"prefix"`
	BID               string  `db:"bid"                 json:"bid"                redis:"bid"`                  //  下发的银行卡或虚拟钱包的ID
	Flag              int     `db:"flag"                json:"flag"               redis:"flag"`                 //  1=银行卡,2=虚拟钱包
	OID               string  `db:"oid"                 json:"oid"                redis:"oid"`                  //  三方ID
	UID               string  `db:"uid"                 json:"uid"                redis:"uid"`                  //
	ParentUID         string  `db:"parent_uid"          json:"parent_uid"         redis:"parent_uid"`           //  上级uid
	ParentName        string  `db:"parent_name"         json:"parent_name"        redis:"parent_name"`          // 上级代理名
	Username          string  `db:"username"            json:"username"           redis:"username"`             //
	PID               string  `db:"pid"                 json:"pid"                redis:"pid"`                  //  paymendID
	Amount            float64 `db:"amount"              json:"amount"             redis:"amount"`               // 提款金额
	State             int     `db:"state"               json:"state"              redis:"state"`                // 371:审核中 372:审核拒绝 373:出款中 374:提款成功 375:出款失败 376:异常订单 377:代付失败
	Automatic         int     `db:"automatic"           json:"automatic"          redis:"automatic"`            // 是否自动出款:0=手工,1=自动
	BankName          string  `db:"bank_name"           json:"bank_name"          redis:"bank_name"`            // 出款卡的银行名称
	RealName          string  `db:"real_name"           json:"real_name"          redis:"real_name"`            // 出款卡的开户人
	CardNo            string  `db:"card_no"             json:"card_no"            redis:"card_no"`              // 出款卡的卡号
	CreatedAt         int64   `db:"created_at"          json:"created_at"         redis:"created_at"`           //
	ConfirmAt         int64   `db:"confirm_at"          json:"confirm_at"         redis:"confirm_at"`           // 确认时间
	ConfirmUID        string  `db:"confirm_uid"         json:"confirm_uid"        redis:"confirm_uid"`          // 手动确认人uid
	ReviewRemark      string  `db:"review_remark"       json:"review_remark"      redis:"review_remark"`        // 审核备注
	WithdrawAt        int64   `db:"withdraw_at"         json:"withdraw_at"        redis:"withdraw_at"`          // 出款时间
	ConfirmName       string  `db:"confirm_name"        json:"confirm_name"       redis:"confirm_name"`         // 手动确认人名字
	WithdrawUID       string  `db:"withdraw_uid"        json:"withdraw_uid"       redis:"withdraw_uid"`         // 出款人的ID
	WithdrawName      string  `db:"withdraw_name"       json:"withdraw_name"      redis:"withdraw_name"`        //  出款人的名字
	WithdrawRemark    string  `db:"withdraw_remark"     json:"withdraw_remark"    redis:"withdraw_remark"`      // 出款备注
	FinanceType       int     `db:"finance_type"        json:"finance_type"       redis:"finance_type"`         // 财务类型 156=提款 165=代客提款 167=代理提款
	LastDepositAmount float64 `db:"last_deposit_amount" json:"last_deposit_amount" redis:"last_deposit_amount"` // 上笔成功存款金额
	RealNameHash      string  `db:"real_name_hash"      json:"real_name_hash"     redis:"real_name_hash"`       // 真实姓名哈希
	HangUpUID         string  `db:"hang_up_uid"         json:"hang_up_uid"        redis:"hang_up_uid"`          // 挂起人uid
	HangUpRemark      string  `db:"hang_up_remark"      json:"hang_up_remark"     redis:"hang_up_remark"`       // 挂起备注
	HangUpName        string  `db:"hang_up_name"        json:"hang_up_name"       redis:"hang_up_name"`         // 挂起人名字
	RemarkID          int64   `db:"remark_id"           json:"remark_id"          redis:"remark_id"`            // 挂起原因ID
	HangUpAt          int64   `db:"hang_up_at"          json:"hang_up_at"         redis:"hang_up_at"`           // 挂起时间
	ReceiveAt         int64   `db:"receive_at"          json:"receive_at"         redis:"receive_at"`           // 领取时间
	WalletFlag        int     `db:"wallet_flag"         json:"wallet_flag"        redis:"wallet_flag"`          // 钱包类型:1=中心钱包,2=佣金钱包
	TopUID            string  `db:"top_uid"             json:"top_uid"            redis:"top_uid"`              // 总代uid
	TopName           string  `db:"top_name"            json:"top_name"           redis:"top_name"`             // 总代用户名
	Level             int     `db:"level"               json:"level"              redis:"level"`
}

// FWithdrawData 取款数据
type FWithdrawData struct {
	T   int64             `json:"t"`
	D   []Withdraw        `json:"d"`
	Agg map[string]string `json:"agg"`
}

type WithdrawData struct {
	D []Withdraw `json:"d"`
	T int64      `json:"t"`
	S uint       `json:"s"`
}

type WithdrawListData struct {
	T   int64             `json:"t"`
	D   []withdrawCols    `json:"d"`
	Agg map[string]string `json:"agg"`
}

type withdrawCols struct {
	Withdraw
	CateID             string  `json:"cate_id"`
	CateName           string  `json:"cate_name"`
	MemberBankName     string  `json:"member_bank_name"`
	MemberBankNo       string  `json:"member_bank_no"`
	MemberBankRealName string  `json:"member_bank_real_name"`
	MemberBankAddress  string  `json:"member_bank_address"`
	MemberRealName     string  `json:"member_real_name"`
	MemberTags         string  `json:"member_tags"`
	Balance            float64 `db:"balance"     json:"balance"     redis:"balance"    ` //余额
	LockAmount         float64 `db:"lock_amount" json:"lock_amount" redis:"lock_amount"` //锁定额度
}

type withdrawTotal struct {
	T   sql.NullInt64   `json:"t"`
	Agg sql.NullFloat64 `json:"agg"`
}

// WithdrawUserInsert 用户申请订单
func WithdrawUserInsert(amount, bid string, fctx *fasthttp.RequestCtx) (string, error) {

	// check member
	member, err := MemberCache(fctx)
	if err != nil {
		return "", err
	}

	var bankcardHash uint64
	query, _, _ := dialect.From("tbl_member_bankcard").Select("bank_card_hash").Where(g.Ex{"id": bid}).ToSQL()
	err = meta.MerchantDB.Get(&bankcardHash, query)
	if err != nil {
		return "", err
	}

	// 记录不存在
	if bankcardHash == 0 {
		return "", errors.New(helper.RecordNotExistErr)
	}

	idx := bankcardHash % 10
	key := fmt.Sprintf("bl:bc%d", idx)
	ok, err := meta.MerchantRedis.SIsMember(ctx, key, bankcardHash).Result()
	if err != nil {
		return "", pushLog(err, helper.RedisErr)
	}

	if ok {
		return "", errors.New(helper.BankcardAbnormal)
	}

	var (
		receiveAt  int64
		withdrawId = helper.GenLongId()
		state      = WithdrawReviewing
		adminName  string
	)

	// 获取风控UID
	uid, err := GetRisksUID()
	if err != nil {
		fmt.Println("风控人员未找到: 订单id=", withdrawId, "err:", err)
		uid = "0"
	}

	if uid != "0" {
		// 获取风控审核人的name
		adminName, err = AdminGetName(uid)
		if err != nil {
			return "", err
		}

		if adminName == "" {
			fmt.Println("风控人员未找到: 订单id=", withdrawId, "uid:", uid, "err:", err)
			uid = "0"
		}

		if uid != "0" {
			state = WithdrawDispatched
			receiveAt = fctx.Time().Unix()
		}
	}

	// 记录提款单
	err = WithdrawInsert(amount, bid, withdrawId, uid, adminName, receiveAt, state, fctx.Time(), member)
	if err != nil {
		return "", err
	}

	if uid != "0" {
		_ = SetRisksOrder(uid, withdrawId, 1)
	} else {
		/*
			// 自动派单模式
			exist, _ := meta.MerchantRedis.Get(ctx, risksState).Result()
			if exist == "1" {
				// 无风控人员可以分配
				param := map[string]interface{}{
					"id": withdrawId,
				}
				_, _ = BeanPut("risk", param, 10)
			}
		*/
	}

	// 发送消息通知
	_ = PushWithdrawNotify(withdrawReviewFmt, member.Username, amount)

	return withdrawId, nil
}

func WithdrawInsert(amount, bid, withdrawID, confirmUid, confirmName string, receiveAt int64, state int, ts time.Time, member Member) error {

	// lock and defer unlock
	lk := fmt.Sprintf("w:%s", member.Username)
	err := Lock(lk)
	if err != nil {
		return err
	}

	defer Unlock(lk)

	// 同时只能有一笔提款在处理中
	ex := g.Ex{
		"uid":   member.UID,
		"state": g.Op{"notIn": []int64{WithdrawReviewReject, WithdrawSuccess, WithdrawFailed}},
	}

	err = withdrawOrderExists(ex)
	if err != nil {
		return err
	}

	// 判断银行卡
	ex = g.Ex{
		"uid":   member.UID,
		"id":    bid,
		"state": 1,
	}
	exist := BankCardExist(ex)
	if !exist {
		return errors.New(helper.BankCardNotExist)
	}

	withdrawAmount, err := decimal.NewFromString(amount)
	if err != nil {
		return pushLog(err, helper.AmountErr)
	}

	// check balance
	userAmount, err := BalanceIsEnough(member.UID, withdrawAmount)
	if err != nil {
		return err
	}

	// 检查当日是否还有提现次数和提现额度
	//date := ts.Format("20060102")
	//flg, err = withdrawLimitCheck(cli, date, member.Username, member.Level, withdrawAmount)
	//if err != nil {
	//	return "", flg, err
	//}

	lastDeposit, err := depositLastAmount(member.UID)
	if err != nil {
		return err
	}

	// 默认取代代付
	//automatic := 1
	//// 根据金额判断 该笔提款是否走代付渠道
	//if withdrawAmount.GreaterThanOrEqual(decimal.NewFromInt32(10000)) {
	//	automatic = 0
	//}
	automatic := 0

	record := g.Record{
		"id":                  withdrawID,
		"prefix":              meta.Prefix,
		"bid":                 bid,
		"flag":                1,
		"oid":                 withdrawID,
		"uid":                 member.UID,
		"top_uid":             member.TopUID,
		"top_name":            member.TopName,
		"parent_name":         member.ParentName,
		"parent_uid":          member.ParentUID,
		"username":            member.Username,
		"pid":                 0,
		"amount":              withdrawAmount.Truncate(4).String(),
		"state":               state,
		"automatic":           automatic, //1:自动转账  0:人工确认
		"created_at":          ts.Unix(),
		"finance_type":        TransactionWithDraw,
		"real_name_hash":      strconv.FormatUint(member.RealnameHash, 10),
		"last_deposit_amount": lastDeposit,
		"receive_at":          receiveAt,
		"confirm_uid":         confirmUid,
		"confirm_name":        confirmName,
		"wallet_flag":         MemberWallet,
		"level":               member.Level,
	}

	// 开启事务 写账变 更新redis  查询提款
	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	query, _, _ := dialect.Insert("tbl_withdraw").Rows(record).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	// 更新余额
	ex = g.Ex{
		"uid":    member.UID,
		"prefix": meta.Prefix,
	}
	balanceRecord := g.Record{
		"balance":     g.L(fmt.Sprintf("balance-%s", withdrawAmount.String())),
		"lock_amount": g.L(fmt.Sprintf("lock_amount+%s", withdrawAmount.String())),
	}
	query, _, _ = dialect.Update("tbl_members").Set(balanceRecord).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	// 写入账变
	mbTrans := memberTransaction{
		AfterAmount:  userAmount.Sub(withdrawAmount).String(),
		Amount:       withdrawAmount.String(),
		BeforeAmount: userAmount.String(),
		BillNo:       withdrawID,
		CreatedAt:    ts.UnixNano() / 1e6,
		ID:           helper.GenId(),
		CashType:     TransactionWithDraw,
		UID:          member.UID,
		Username:     member.Username,
		Prefix:       meta.Prefix,
	}

	query, _, _ = dialect.Insert("tbl_balance_transaction").Rows(mbTrans).ToSQL()
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

// 检查订单是否存在
func withdrawOrderExists(ex g.Ex) error {

	var id string
	query, _, _ := dialect.From("tbl_withdraw").Select("id").Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&id, query)

	if err != nil && err != sql.ErrNoRows {
		return pushLog(err, helper.DBErr)
	}

	if id != "" {
		return errors.New(helper.OrderProcess)
	}

	return nil
}

func BankCardExist(ex g.Ex) bool {

	var id string
	t := dialect.From("tbl_member_bankcard")
	query, _, _ := t.Select("uid").Where(ex).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&id, query)
	if err == sql.ErrNoRows {
		return false
	}

	return true
}

// WithdrawList 提款记录
func WithdrawList(ex g.Ex, ty uint8, startTime, endTime string, page, pageSize uint) (FWithdrawData, error) {

	ex["prefix"] = meta.Prefix

	if startTime != "" && endTime != "" {

		startAt, err := helper.TimeToLoc(startTime, loc)
		if err != nil {
			return FWithdrawData{}, errors.New(helper.DateTimeErr)
		}

		endAt, err := helper.TimeToLoc(endTime, loc)
		if err != nil {
			return FWithdrawData{}, errors.New(helper.DateTimeErr)
		}

		if ty == 1 {
			ex["created_at"] = g.Op{"between": exp.NewRangeVal(startAt, endAt)}
		} else {
			ex["withdraw_at"] = g.Op{"between": exp.NewRangeVal(startAt, endAt)}
		}
	}
	// 待派单特殊操作 只显示一天的数据
	if ty == 3 {
		now := time.Now().Unix()
		ex["created_at"] = g.Op{"between": exp.NewRangeVal(now-172800, now)}
	}

	if realName, ok := ex["real_name_hash"]; ok {
		ex["real_name_hash"] = MurmurHash(realName.(string), 0)
	}

	var data FWithdrawData
	if page == 1 {
		var total withdrawTotal
		query, _, _ := dialect.From("tbl_withdraw").Select(g.COUNT(1).As("t"), g.SUM("amount").As("agg")).Where(ex).ToSQL()
		err := meta.MerchantDB.Get(&total, query)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}

		data.T = total.T.Int64
		data.Agg = map[string]string{
			"amount": decimal.NewFromFloat(total.Agg.Float64).Truncate(4).String(),
		}
	}

	offset := (page - 1) * pageSize
	query, _, _ := dialect.From("tbl_withdraw").
		Select(colWithdraw...).Where(ex).Order(g.C("created_at").Desc()).Offset(offset).Limit(pageSize).ToSQL()
	err := meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}

// 提款历史记录
func WithdrawHistoryList(ex g.Ex, rangeParam map[string][]interface{}, ty uint8, startTime, endTime string, page, pageSize uint) (FWithdrawData, error) {

	ex["prefix"] = meta.Prefix
	if startTime != "" && endTime != "" {

		startAt, err := helper.TimeToLoc(startTime, loc)
		if err != nil {
			return FWithdrawData{}, errors.New(helper.DateTimeErr)
		}

		endAt, err := helper.TimeToLoc(endTime, loc)
		if err != nil {
			return FWithdrawData{}, errors.New(helper.DateTimeErr)
		}

		if startAt >= endAt {
			return FWithdrawData{}, errors.New(helper.QueryTimeRangeErr)
		}

		if ty == 1 {
			rangeParam["created_at"] = []interface{}{startAt, endAt}
		} else {
			rangeParam["withdraw_at"] = []interface{}{startAt, endAt}
		}
	}

	if realName, ok := ex["real_name_hash"]; ok {
		ex["real_name_hash"] = MurmurHash(realName.(string), 0)
	}

	aggField := map[string]string{
		"amount": "amount",
	}

	data := FWithdrawData{Agg: map[string]string{}}
	total, esData, aggData, err := EsSearch(
		esPrefixIndex("tbl_withdraw"),
		"created_at",
		int(page),
		int(pageSize), withdrawFields, ex, rangeParam, aggField)
	if err != nil {
		return data, err
	}

	for k, v := range aggField {
		amount, _ := aggData.Sum(k)
		if amount != nil {
			data.Agg[v] = fmt.Sprintf("%.4f", *amount.Value)
		}
	}

	data.T = total
	for _, v := range esData {
		withdraw := Withdraw{}
		withdraw.ID = v.Id
		err = helper.JsonUnmarshal(v.Source, &withdraw)
		if err != nil {
			return data, errors.New(helper.FormatErr)
		}
		data.D = append(data.D, withdraw)
	}

	return data, nil
}

// 财务拒绝提款订单
func WithdrawReject(id string, d g.Record) error {
	return WithdrawDownPoint(id, "", WithdrawFailed, d)
}

// 人工出款成功
func WithdrawHandSuccess(id, uid, bid string, record g.Record) error {

	bankcard, err := withdrawGetBankcard(uid, bid)
	if err != nil {
		fmt.Println("query bankcard error: ", err.Error())
	}

	return WithdrawDownPoint(id, bankcard, WithdrawSuccess, record)
}

func WithdrawRiskReview(id string, state int, record g.Record, withdraw Withdraw) error {

	err := WithdrawDownPoint(id, "", state, record)
	if err != nil {
		return err
	}

	var confirmUID string
	ex := g.Ex{
		"id": id,
	}
	query, _, _ := dialect.From("tbl_withdraw").Select("confirm_uid").Where(ex).Limit(1).ToSQL()
	err = meta.MerchantDB.Get(&confirmUID, query)
	if err != nil {
		fmt.Println(pushLog(err, helper.DBErr))
	}

	if confirmUID != "" && confirmUID != withdraw.ConfirmUID {
		_ = SetRisksOrder(confirmUID, id, -1)
	}

	_ = SetRisksOrder(withdraw.ConfirmUID, id, -1)

	return nil
}

// WithdrawHandToAuto 手动代付
func WithdrawHandToAuto(uid, username, id, pid, bid string, amount float64, t time.Time) error {

	bankcard, err := WithdrawGetBank(bid, username)
	if err != nil {
		return err
	}

	// query realName and bankcardNo
	bankcardNo, realName, err := WithdrawGetBkAndRn(bid, uid, false)
	if err != nil {
		return err
	}

	p, err := ChanWithdrawByCateID(pid)
	if err != nil {
		return err
	}

	if len(p.ID) == 0 || p.State == "0" {
		return errors.New(helper.CateNotExist)
	}

	pay, err := WithdrawGetPayment(p.CateID)
	if err != nil {
		return err
	}

	as := strconv.FormatFloat(amount, 'f', -1, 64)
	// check amount range, continue the for loop if amount out of range
	_, ok := validator.CheckFloatScope(as, p.Fmin, p.Fmax)
	if !ok {
		return errors.New(helper.AmountOutRange)
	}

	bank, err := withdrawMatchBank(p.ID, bankcard.BankID)
	if err != nil {
		return err
	}

	kvnd := decimal.NewFromInt(1000)
	param := WithdrawAutoParam{
		OrderID:    id,
		Amount:     decimal.NewFromFloat(amount).Mul(kvnd).String(),
		BankID:     bank.ID,
		BankCode:   bank.Code,
		CardNumber: bankcardNo, // 银行卡号
		CardName:   realName,   // 持卡人姓名
		Ts:         t,          // 时间
		PaymentID:  p.ID,
	}

	// param.BankCode = bank.Code
	oid, err := Withdrawal(pay, param)
	if err != nil {
		fmt.Println("withdrawHandToAuto failed 1:", id, err)
		return err
	}

	err = withdrawAutoUpdate(param.OrderID, oid, bank.PaymentID, WithdrawDealing)
	if err != nil {
		fmt.Println("withdrawHandToAuto failed 2:", id, err)
	}

	return nil
}

// match bank and channel, if matched return the bank information
func withdrawMatchBank(pid, bid string) (Bank_t, error) {

	bank := Bank_t{}
	ex := g.Ex{
		"payment_id": pid,
		"bank_id":    bid,
		"state":      "1",
		"prefix":     meta.Prefix,
	}
	query, _, _ := dialect.From("f_channel_banks").Select(colChannelBank...).Where(ex).Limit(1).ToSQL()
	fmt.Println(query)
	err := meta.MerchantDB.Get(&bank, query)
	if err != nil {
		return bank, pushLog(err, helper.DBErr)
	}
	// check bank, continue the for loop if bank not supported
	//res, err := meta.MerchantRedis.Get(ctx, "BK:"+pid).Result()
	//if err != nil {
	//	fmt.Println(err)
	//	return bank, pushLog(err, helper.RedisErr)
	//}
	//
	//// unmarshal the searched result of string of the banks to the destination []Bank_t struct
	//var banks []Bank_t
	//err = helper.JsonUnmarshal([]byte(res), &banks)
	//if err != nil {
	//	fmt.Println(err)
	//	return bank, errors.New(helper.FormatErr)
	//}
	//
	//// for loop the banks to match the suitable bank
	//for _, v := range banks {
	//	if v.BankID != bid {
	//		continue
	//	}
	//	return v, nil
	//}

	return bank, nil
}

// WithdrawAutoPaySetFailed 将订单状态从出款中修改为代付失败
func WithdrawAutoPaySetFailed(id string, confirmAt int64, confirmUid, confirmName string) error {

	order, err := WithdrawFind(id)
	if err != nil {
		return err
	}

	// 只能将出款中(三方处理中, 即自动代付)的订单状态流转为代付失败
	if order.State != WithdrawDealing || order.Automatic != 1 {
		return errors.New(helper.OrderStateErr)
	}

	// 将automatic设为1是为了确保状态为代付失败的订单一定为自动出款(automatic=1)
	record := g.Record{
		"state":        WithdrawAutoPayFailed,
		"automatic":    "1",
		"confirm_at":   confirmAt,
		"confirm_uid":  confirmUid,
		"confirm_name": confirmName,
	}

	return WithdrawDownPoint(id, "", WithdrawAutoPayFailed, record)
}

// WithdrawFind 查找单条提款记录, 订单不存在返回错误: OrderNotExist
func WithdrawFind(id string) (Withdraw, error) {

	w := Withdraw{}
	query, _, _ := dialect.From("tbl_withdraw").Select(colWithdraw...).Where(g.Ex{"id": id}).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&w, query)
	if err == sql.ErrNoRows {
		return w, errors.New(helper.OrderNotExist)
	}

	if err != nil {
		return w, pushLog(err, helper.DBErr)
	}

	return w, nil
}

// 每日剩余提款次数和总额
func WithdrawLimit(ctx *fasthttp.RequestCtx) (map[string]string, error) {

	member, err := MemberCache(ctx)
	if err != nil {
		return nil, err
	}

	date := ctx.Time().Format("20060102")
	return WithDrawDailyLimit(date, member.Username)
}

// WithDrawDailyLimit 获取每日提现限制
func WithDrawDailyLimit(date, username string) (map[string]string, error) {

	limitKey := fmt.Sprintf("%s:w:%s", username, date)

	// 获取会员当日提现限制，先取缓存 没有就设置一下
	num, err := meta.MerchantRedis.Exists(ctx, limitKey).Result()
	if err != nil {
		return defaultLevelWithdrawLimit, pushLog(err, helper.RedisErr)
	}

	if num > 0 {
		rs, err := meta.MerchantRedis.Get(ctx, limitKey).Result()
		if err != nil {
			return defaultLevelWithdrawLimit, pushLog(err, helper.RedisErr)
		}

		data := make(map[string]string)
		err = helper.JsonUnmarshal([]byte(rs), &data)
		if err != nil {
			return defaultLevelWithdrawLimit, errors.New(helper.FormatErr)
		}

		return data, nil
	}

	b, _ := helper.JsonMarshal(defaultLevelWithdrawLimit)
	err = meta.MerchantRedis.Set(ctx, limitKey, b, 24*60*60*time.Second).Err()
	if err != nil {
		return defaultLevelWithdrawLimit, pushLog(err, helper.RedisErr)
	}

	return defaultLevelWithdrawLimit, nil
}

func WithdrawInProcessing(ctx *fasthttp.RequestCtx) (map[string]interface{}, error) {

	var data map[string]interface{}

	member, err := MemberCache(ctx)
	if err != nil {
		return data, err
	}

	order := Withdraw{}
	ex := g.Ex{
		"uid":   member.UID,
		"state": g.Op{"notIn": []int64{WithdrawReviewReject, WithdrawSuccess, WithdrawFailed}},
	}
	query, _, _ := dialect.From("tbl_withdraw").Select(colsWithdraw...).Where(ex).Limit(1).ToSQL()
	err = meta.MerchantDB.Get(&order, query)
	if err != nil && err != sql.ErrNoRows {
		return nil, pushLog(err, helper.DBErr)
	}

	if err == sql.ErrNoRows {
		return nil, nil
	}

	data = map[string]interface{}{
		"id":         order.ID,
		"bid":        order.BID,
		"amount":     order.Amount,
		"state":      order.State,
		"created_at": order.CreatedAt,
	}

	return data, nil
}

// 处理 提款订单返回数据
func WithdrawDealListData(data FWithdrawData) (WithdrawListData, error) {

	result := WithdrawListData{
		T:   data.T,
		Agg: data.Agg,
	}

	if len(data.D) == 0 {
		return result, nil
	}

	// 获取渠道号的pid slice
	pids := make([]string, 0)
	var agencyNames []string
	// 组装获取rpc数据参数
	rpcParam := make(map[string][]string)
	namesMap := make(map[string]string)
	for _, v := range data.D {
		rpcParam["bankcard"] = append(rpcParam["bankcard"], v.BID)
		rpcParam["realname"] = append(rpcParam["realname"], v.UID)
		namesMap[v.Username] = v.UID
		pids = append(pids, v.PID)

		if v.ParentName != "" && v.ParentName != "root" {
			agencyNames = append(agencyNames, v.ParentName)
		}

	}
	userMap := map[string]MBBalance{}
	if len(data.D) > 0 {
		var uids []string

		for _, v := range data.D {
			uids = append(uids, v.UID)
		}

		balances, err := getBalanceByUids(uids)
		if err != nil {
			return result, err
		}

		for _, v := range balances {
			userMap[v.UID] = v
		}

	}

	// 遍历用户map 读取标签数据
	var names []string
	tags := make(map[string]string)
	for name, uid := range namesMap {
		// 获取用户标签
		memberTag, err := MemberTagsList(uid)
		if err != nil {
			return result, err
		}
		// 组装需要通过name获取的 redis参数
		names = append(names, name)
		tags[name] = memberTag
	}

	bankcards, err := bankcardListDBByIDs(rpcParam["bankcard"])
	if err != nil {
		return result, err
	}

	encFields := []string{"realname"}

	for _, v := range rpcParam["bankcard"] {
		encFields = append(encFields, "bankcard"+v)
	}

	recs, err := grpc_t.DecryptAll(rpcParam["realname"], true, encFields)
	if err != nil {
		return result, errors.New(helper.GetRPCErr)
	}

	cids, _ := channelCateMap(pids)

	// 处理返回前端的数据
	for k, v := range data.D {

		fmt.Println("k = ", k)
		w := withdrawCols{
			Withdraw:           v,
			MemberBankNo:       recs[v.UID]["bankcard"+v.BID],
			MemberBankRealName: recs[v.UID]["realname"],
			MemberRealName:     recs[v.UID]["realname"],
			MemberTags:         tags[v.Username],
			Balance:            userMap[v.UID].Balance,
			LockAmount:         userMap[v.UID].LockAmount,
		}

		// 匹配银行卡信息
		card, ok := bankcards[v.BID]
		if ok {
			w.MemberBankName = card.BankID
			w.MemberBankAddress = card.BankAddress
		}

		// 匹配渠道信息
		cate, ok := cids[v.PID]
		if ok {
			w.CateID = cate.ID
			w.CateName = cate.Name
		}

		result.D = append(result.D, w)
	}

	return result, nil
}

func MemberTagsList(uid string) (string, error) {

	var tags []string
	ex := g.Ex{"uid": uid}
	query, _, _ := dialect.From("tbl_member_tags").Select("tag_name").Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&tags, query)
	if err != nil {
		return "", pushLog(err, helper.DBErr)
	}

	return strings.Join(tags, ","), nil
}

func WithdrawUpdateInfo(id string, record g.Record) error {

	query, _, _ := dialect.Update("tbl_withdraw").Where(g.Ex{"id": id}).Set(record).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return nil
}

func WithdrawAuto(param WithdrawAutoParam, level int) error {

	i := 0
	key := "pw:" + strconv.Itoa(level)

	pwc, err := meta.MerchantRedis.LLen(ctx, key).Result()
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}

	for {
		i++

		// the maximum loop times is the length of the list
		if pwc < int64(i) {
			fmt.Printf("withdrawAuto failed 1: %v \n", param)
			return errors.New(helper.NoPayChannel)
		}

		// search payment channel for member by member's level
		res, err := meta.MerchantRedis.RPopLPush(ctx, key, key).Result()
		if err != nil {
			fmt.Println("withdrawAuto failed 2:", param, err)
			continue
		}

		// unmarshal the searched result of string of the payment channel to the destination vip_t struct
		var info Vip_t
		err = helper.JsonUnmarshal([]byte(res), &info)
		if err != nil {
			fmt.Println("withdrawAuto failed 3:", param, err)
			continue
		}

		pay, err := WithdrawGetPayment(info.CateID)
		if err != nil {
			continue
		}

		// amount must be divided by 1000, because the unit of fmin and fmax is k
		amount, _ := decimal.NewFromString(param.Amount)
		amount = amount.Div(decimal.NewFromInt(1000))

		// check amount range, continue the for loop if amount out of range
		_, ok := validator.CheckFloatScope(amount.String(), info.Fmin, info.Fmax)
		if !ok {
			fmt.Println("withdrawAuto failed 4:", param, amount.String(), info.Fmin, info.Fmax)
			continue
		}

		bank, err := withdrawMatchBank(info.PaymentID, param.BankID)
		if err != nil {
			fmt.Println("withdrawAuto failed 5:", param, err)
			continue
		}
		fmt.Println(bank)
		param.BankCode = bank.Code
		param.PaymentID = info.PaymentID
		oid, err := Withdrawal(pay, param)
		if err != nil {
			fmt.Println("withdrawAuto failed 6:", param, err)
			return err
		}

		_ = withdrawAutoUpdate(param.OrderID, oid, info.PaymentID, WithdrawDealing)
		return nil
	}
}

func withdrawAutoUpdate(id, oid, pid string, state int) error {

	r := g.Record{"state": state, "automatic": "1"}
	if oid != "" {
		r["oid"] = oid
	}
	if pid != "" {
		r["pid"] = pid
	}

	query, _, _ := dialect.Update("tbl_withdraw").Set(r).Where(g.Ex{"id": id}).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return err
}

// query bankcard information from redis cache with primary key id of table
// tbl_member_bankcard(which is bid) and the username as parameters
func WithdrawGetBank(bid, username string) (MemberBankCard, error) {

	bank := MemberBankCard{}
	banks, err := MemberBankcardList(g.Ex{"username": username})
	if err != nil && err != sql.ErrNoRows {
		return bank, err
	}

	for _, v := range banks {
		if v.ID == bid {
			bank = v
			break
		}
	}
	if bank.ID != bid {
		return bank, errors.New(helper.BankcardIDErr)
	}

	return bank, nil
}

// query bankcard and realname through by rpc call with the primary key id of table
// tbl_member_bankcard(which is bid) and the username as parameters
func WithdrawGetBkAndRn(bid, uid string, hide bool) (string, string, error) {

	field := "bankcard" + bid
	recs, err := grpc_t.Decrypt(uid, hide, []string{"realname", field})

	if err != nil {
		return "", "", errors.New(helper.GetRPCErr)
	}

	return recs[field], recs["realname"], nil
}

func withdrawGetBankcard(id, bid string) (string, error) {

	field := "bankcard" + bid
	recs, err := grpc_t.Decrypt(id, true, []string{field})
	if err != nil {
		return "", errors.New(helper.GetRPCErr)
	}

	return recs[field], nil
}

// 获取银行卡成功失败的次数
func WithdrawBanKCardNumber(bid string) (int, int) {

	query := elastic.NewBoolQuery().Must(elastic.NewTermQuery("bid", bid), elastic.NewTermQuery("prefix", meta.Prefix))
	aggParam := map[string]*elastic.TermsAggregation{"state": elastic.NewTermsAggregation().Field("state")}

	fsc := elastic.NewFetchSourceContext(true)
	esService := meta.ES.Search().FetchSourceContext(fsc).Query(query).Size(0)
	for k, v := range aggParam {
		esService = esService.Aggregation(k, v)
	}
	resOrder, err := esService.Index(esPrefixIndex("tbl_withdraw")).Do(ctx)
	if err != nil {
		return 0, 0
	}

	agg, ok := resOrder.Aggregations.Terms("state")
	if !ok {
		return 0, 0
	}

	var (
		success int
		fail    int
	)
	for _, v := range agg.Buckets {
		if WithdrawSuccess == int(v.Key.(float64)) {
			success += int(v.DocCount)
		}

		if WithdrawReviewReject == int(v.Key.(float64)) || WithdrawAbnormal == int(v.Key.(float64)) || WithdrawFailed == int(v.Key.(float64)) {
			fail += int(v.DocCount)
		}
	}

	return success, fail
}

func bankcardListDBByIDs(ids []string) (map[string]MemberBankCard, error) {

	data := make(map[string]MemberBankCard)
	if len(ids) == 0 {
		return nil, errors.New(helper.UsernameErr)
	}

	ex := g.Ex{"id": ids}
	bankcards, err := MemberBankcardList(ex)
	if err != nil {
		return data, err
	}

	for _, v := range bankcards {
		data[v.ID] = v
	}

	return data, nil
}

// WithdrawLock 锁定提款订单
// 订单因为外部因素(接口)导致的状态流转应该加锁
func WithdrawLock(id string) error {

	key := fmt.Sprintf(withdrawOrderLockKey, id)
	err := Lock(key)
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}

	return nil
}

// WithdrawUnLock 解锁提款订单
func WithdrawUnLock(id string) {
	Unlock(fmt.Sprintf(withdrawOrderLockKey, id))
}

// 取款下分
func WithdrawDownPoint(did, bankcard string, state int, record g.Record) error {

	//判断状态是否合法
	allow := map[int]bool{
		WithdrawReviewReject:  true,
		WithdrawDealing:       true,
		WithdrawSuccess:       true,
		WithdrawFailed:        true,
		WithdrawAutoPayFailed: true,
	}
	if _, ok := allow[state]; !ok {
		return errors.New(helper.StateParamErr)
	}

	//1、判断订单是否存在
	var order Withdraw
	ex := g.Ex{"id": did}
	query, _, _ := dialect.From("tbl_withdraw").Select(colsWithdraw...).Where(ex).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&order, query)
	if err != nil || len(order.Username) < 1 {
		return errors.New(helper.IDErr)
	}

	query, _, _ = dialect.Update("tbl_withdraw").Set(record).Where(ex).ToSQL()
	switch order.State {
	case WithdrawReviewing:
		// 审核中(风控待领取)的订单只能流向分配, 上层业务处理
		return errors.New(helper.OrderStateErr)
	case WithdrawDealing:
		// 出款处理中可以是自动代付失败(WithdrawAutoPayFailed) 提款成功(WithdrawSuccess) 提款失败(WithdrawFailed)
		// 自动代付失败和提款成功是调用三方代付才会有的状态
		// 提款失败通过提款管理的[拒绝]操作进行流转(同时出款类型必须是手动出款)
		if state == WithdrawAutoPayFailed {
			_, err = meta.MerchantDB.Exec(query)
			if err != nil {
				return pushLog(err, helper.DBErr)
			}

			return nil
		}

		if state != WithdrawSuccess && (state != WithdrawFailed && order.Automatic == 0) {
			return errors.New(helper.OrderStateErr)
		}

	case WithdrawAbnormal:
		// todo 产品说暂时没有这个状态
		return errors.New(helper.OrderStateErr)

	case WithdrawAutoPayFailed:
		// 代付失败可以通过手动代付将状态流转至出款中
		if state == WithdrawDealing {
			_, err = meta.MerchantDB.Exec(query)
			if err != nil {
				return pushLog(err, helper.DBErr)
			}

			return nil
		}

		// 代付失败的订单也可以通过手动出款直接将状态流转至出款成功
		// 代付失败的订单还可以通过拒绝直接将状态流转至出款失败
		if state != WithdrawFailed && state != WithdrawSuccess {
			return errors.New(helper.OrderStateErr)
		}

	case WithdrawHangup:
		// 挂起的订单只能领取(该状态流转上传业务已经处理), 该状态只能流转至审核中(WithdrawReviewing)
		return errors.New(helper.OrderStateErr)

	case WithdrawDispatched:
		// 派单状态可流转状态为 挂起(WithdrawHangup) 通过(WithdrawDealing) 拒绝(WithdrawReviewReject)
		// 其中流转至挂起状态由上层业务处理
		if state == WithdrawDealing {
			_, err = meta.MerchantDB.Exec(query)
			if err != nil {
				return pushLog(err, helper.DBErr)
			}

			return nil
		}

		if state != WithdrawReviewReject {
			return errors.New(helper.OrderStateErr)
		}

	default:
		// 审核拒绝, 提款成功, 出款失败三个状态为终态 不能进行其他处理
		return errors.New(helper.OrderStateErr)
	}

	// 3 如果是出款成功,修改订单状态为提款成功,扣除锁定钱包中的钱,发送通知
	if WithdrawSuccess == state {
		return withdrawOrderSuccess(query, bankcard, order)
	}

	// 出款失败
	return withdrawOrderFailed(query, order)
}

// 检查锁定钱包余额是否充足
func LockBalanceIsEnough(uid string, amount decimal.Decimal) (decimal.Decimal, error) {

	balance, err := GetBalanceDB(uid)
	if err != nil {
		return decimal.NewFromFloat(balance.LockAmount), err
	}
	if decimal.NewFromFloat(balance.LockAmount).Sub(amount).IsNegative() {
		return decimal.NewFromFloat(balance.LockAmount), errors.New(helper.LackOfBalance)
	}

	return decimal.NewFromFloat(balance.LockAmount), nil
}

// 提款成功
func withdrawOrderSuccess(query, bankcard string, order Withdraw) error {

	money := decimal.NewFromFloat(order.Amount)

	// 判断锁定余额是否充足
	_, err := LockBalanceIsEnough(order.UID, money)
	if err != nil {
		return err
	}

	//开启事务
	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	// 更新提款订单状态
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	// 锁定钱包下分
	ex := g.Ex{
		"uid":    order.UID,
		"prefix": meta.Prefix,
	}
	gr := g.Record{
		"lock_amount": g.L(fmt.Sprintf("lock_amount-%s", money.String())),
	}
	query, _, _ = dialect.Update("tbl_members").Set(gr).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	err = tx.Commit()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}
	// 发送通知 提款成功
	_ = PushWithdrawSuccess(order.UID, order.Amount)

	// 修改会员提款限制
	date := time.Unix(order.CreatedAt, 0).Format("20060102")
	_ = withDrawDailyLimitUpdate(money, date, order.Username)
	return nil
}

// 更新每日提款次数限制和金额限制
func withDrawDailyLimitUpdate(amount decimal.Decimal, date, username string) error {

	limitKey := fmt.Sprintf("%s:w:%s", username, date)

	var wl = map[string]string{
		"withdraw_count": "0",
		"count_remain":   "0",
		"withdraw_max":   "0.0000",
		"max_remain":     "0.0000",
	}

	// 如果订单生成日的提款限制缓存没有命中 从redis中get key的时 返回的err不会是nil
	// 所以直接返回就可以了
	// 否则需要刷新一下缓存
	rs, err := meta.MerchantRedis.Get(ctx, limitKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return pushLog(err, helper.RedisErr)
	}

	err = helper.JsonUnmarshal([]byte(rs), &wl)
	if err != nil {
		return errors.New(helper.FormatErr)
	}

	count, _ := strconv.ParseInt(wl["count_remain"], 10, 64)
	wl["count_remain"] = strconv.FormatInt(count-1, 10)

	prev, _ := decimal.NewFromString(wl["max_remain"])
	wl["max_remain"] = prev.Sub(amount).String()

	b, _ := helper.JsonMarshal(wl)
	err = meta.MerchantRedis.Set(ctx, limitKey, b, 24*60*60*time.Second).Err()
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}

	return nil
}

func withdrawOrderFailed(query string, order Withdraw) error {

	money := decimal.NewFromFloat(order.Amount)

	//4、查询用户额度
	balance, err := GetBalanceDB(order.UID)
	if err != nil {
		return err
	}
	balanceAfter := decimal.NewFromFloat(balance.Balance).Add(money)

	//开启事务
	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	//5、更新订单状态
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	//6、更新余额
	ex := g.Ex{
		"uid":    order.UID,
		"prefix": meta.Prefix,
	}
	balanceRecord := g.Record{
		"balance":     g.L(fmt.Sprintf("balance+%s", money.String())),
		"lock_amount": g.L(fmt.Sprintf("lock_amount-%s", money.String())),
	}
	query, _, _ = dialect.Update("tbl_members").Set(balanceRecord).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.DBErr)
	}

	//7、新增账变记录
	mbTrans := memberTransaction{
		AfterAmount:  balanceAfter.String(),
		Amount:       money.String(),
		BeforeAmount: decimal.NewFromFloat(balance.Balance).String(),
		BillNo:       order.ID,
		CreatedAt:    time.Now().UnixNano() / 1e6,
		ID:           helper.GenId(),
		CashType:     TransactionWithDrawFail,
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

	err = tx.Commit()
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return nil
}

// 接收到三方回调后调用这个方法（三方调用缺少confirm uid和confirm name这些信息）
func withdrawUpdate(id, uid, bid string, state int, t time.Time) error {

	// 加锁
	err := withdrawLock(id)
	if err != nil {
		return err
	}
	defer withdrawUnLock(id)

	record := g.Record{
		"state": state,
	}

	switch state {
	case WithdrawSuccess:
		record["automatic"] = "1"
		record["withdraw_at"] = fmt.Sprintf("%d", t.Unix())
	case WithdrawAutoPayFailed:
		record["confirm_at"] = fmt.Sprintf("%d", t.Unix())
	default:
		return errors.New(helper.StateParamErr)
	}

	bankcard, err := withdrawGetBankcard(uid, bid)
	if err != nil {
		fmt.Println("query bankcard error: ", err.Error())
	}

	return WithdrawDownPoint(id, bankcard, state, record)
}

// WithdrawLock 锁定提款订单
// 订单因为外部因素(接口)导致的状态流转应该加锁
func withdrawLock(id string) error {

	key := fmt.Sprintf(withdrawOrderLockKey, id)
	return Lock(key)
}

// WithdrawUnLock 解锁提款订单
func withdrawUnLock(id string) {

	key := fmt.Sprintf(withdrawOrderLockKey, id)
	Unlock(key)
}

func withdrawUpdateInfo(ex g.Ex, record g.Record) error {
	ex["prefix"] = meta.Prefix
	query, _, _ := dialect.Update("tbl_withdraw").Set(record).Where(ex).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	return err
}

// 查找单条提款记录, 订单不存在返回错误: OrderNotExist
func withdrawFind(id string) (Withdraw, error) {

	w := Withdraw{}
	query, _, _ := dialect.From("tbl_withdraw").Select(colWithdraw...).Where(g.Ex{"id": id}).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&w, query)
	fmt.Println(query)
	if err == sql.ErrNoRows {
		return w, errors.New(helper.OrderNotExist)
	}

	if err != nil {
		return w, err
	}

	return w, nil
}
