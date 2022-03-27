package model

import (
	"finance/contrib/helper"
	"finance/contrib/validator"
	"fmt"
	"strconv"
	"time"

	g "github.com/doug-martin/goqu/v9"
	"github.com/shopspring/decimal"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
	"github.com/wI2L/jettison"
)

// NewestPay 调用与pid对应的渠道, 发起充值(代付)请求
func NewestPay(ctx *fasthttp.RequestCtx, pid, amount, bid string, user Member) {

	var (
		err  error
		data paymentDepositResp
	)
	pLog := &paymentTDLog{
		Lable:    paymentLogTag,
		Flag:     "deposit",
		Username: user.Username,
	}
	// 记录请求日志
	defer func() {
		if err != nil {
			pLog.Error = fmt.Sprintf("{req: %s, err: %s}", ctx.PostArgs().String(), err.Error())
		}
		paymentPushLog(pLog)
	}()

	p, err := CachePayment(pid)
	if err != nil {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	data, err = Pay(pLog, user, p, amount, bid)
	if err != nil {
		// 兼容日志写入 如果输出的不是标准提示 则使用标准提示输出 记录原始错误信息
		if _, e := strconv.Atoi(err.Error()); e == nil {
			helper.Print(ctx, false, err.Error())
			return
		}

		helper.Print(ctx, false, helper.ChannelBusyTryOthers)
		return
	}

	d := g.Record{
		"id":                pLog.OrderID,
		"prefix":            meta.Prefix,
		"oid":               data.OrderID,
		"uid":               user.UID,
		"top_uid":           user.TopUID,
		"top_name":          user.TopName,
		"parent_name":       user.ParentName,
		"parent_uid":        user.ParentUID,
		"username":          user.Username,
		"channel_id":        p.ChannelID,
		"level":             user.Level,
		"cid":               p.CateID,
		"pid":               p.ID,
		"amount":            amount,
		"usdt_apply_amount": 0,
		"rate":              1,
		"state":             DepositConfirming,
		"finance_type":      TransactionDeposit,
		"automatic":         "1",
		"created_at":        fmt.Sprintf("%d", ctx.Time().Unix()),
		"created_uid":       "0",
		"created_name":      "",
		"confirm_at":        "0",
		"confirm_uid":       "0",
		"confirm_name":      "",
		"review_remark":     "",
		"protocol_type":     "",
		"address":           "",
		"flag":              DepositFlagThird,
	}

	// 请求成功插入订单
	err = deposit(d)
	if err != nil {
		pLog.Error = fmt.Sprintf("insert into table error: [%v]", err)
		helper.Print(ctx, false, helper.DBErr)
		return
	}

	// 记录存款行为
	_ = cacheDepositProcessingInsert(user.UID, pLog.OrderID, ctx.Time().Unix())

	res := payCommRes{
		ID:  pLog.OrderID,
		URL: data.Addr,
	}

	helper.Print(ctx, true, res)
}

// CoinPay 调用与pid对应的渠道, 发起充值(代付)请求  amount 越南盾金额
func CoinPay(ctx *fasthttp.RequestCtx, pid, amount string, user Member) {

	var (
		err  error
		data paymentDepositResp
	)
	pLog := &paymentTDLog{
		Lable:    paymentLogTag,
		Flag:     "deposit",
		Username: user.Username,
	}
	// 记录请求日志
	defer func() {
		if err != nil {
			pLog.Error = fmt.Sprintf("{req: %s, err: %s}", ctx.PostArgs().String(), err.Error())
		}
		paymentPushLog(pLog)
	}()

	protocolType := string(ctx.PostArgs().Peek("protocol_type"))
	if protocolType != "TRC20" && protocolType != "ERC20" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	p, err := CachePayment(pid)
	if err != nil {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	rate, err := USDTConfig()
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	dm, err := decimal.NewFromString(amount)
	if err != nil {
		helper.Print(ctx, false, helper.AmountErr)
		return
	}

	// 提交usdt金额给三方
	usdtAmount := dm.Mul(decimal.NewFromInt(1000)).DivRound(rate, 3).String()

	payment, ok := paymentRoute[p.CateID]
	if !ok {
		helper.Print(ctx, false, helper.NoPayChannel)
		return
	}

	ch := paymentChannelMatch(p.ChannelID)
	pLog.Merchant = payment.Name()
	pLog.Channel = string(ch)

	// 生成我方存款订单号
	pLog.OrderID = helper.GenId()

	// 检查用户的存款行为是否过于频繁
	err = cacheDepositProcessing(user.UID, time.Now().Unix())
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	// 向渠道方发送存款订单请求
	data, err = payment.Pay(pLog, ch, usdtAmount, "")
	if err != nil {
		// 兼容日志写入 如果输出的不是标准提示 则使用标准提示输出 记录原始错误信息
		if _, e := strconv.Atoi(err.Error()); e == nil {
			helper.Print(ctx, false, err.Error())
			return
		}

		helper.Print(ctx, false, helper.ChannelBusyTryOthers)
		return
	}

	v, err := fastjson.Parse(data.Addr)
	if err != nil {
		helper.Print(ctx, false, helper.FormatErr)
		return
	}

	d := g.Record{
		"id":                pLog.OrderID,
		"prefix":            meta.Prefix,
		"oid":               data.OrderID,
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
		"usdt_apply_amount": usdtAmount,
		"rate":              rate.String(),
		"state":             DepositConfirming,
		"finance_type":      TransactionDeposit,
		"automatic":         "1",
		"created_at":        fmt.Sprintf("%d", ctx.Time().Unix()),
		"created_uid":       "0",
		"created_name":      "",
		"confirm_at":        "0",
		"confirm_uid":       "0",
		"confirm_name":      "",
		"review_remark":     "",
		"protocol_type":     protocolType,
		"address":           string(v.GetStringBytes(protocolType)),
		"flag":              DepositFlagThirdUSTD,
		"level":             user.Level,
	}

	// 请求成功插入订单
	err = deposit(d)
	if err != nil {
		pLog.Error = fmt.Sprintf("insert into table error: [%v]", err)
		helper.Print(ctx, false, helper.DBErr)
		return
	}

	// 记录存款行为
	_ = cacheDepositProcessingInsert(user.UID, pLog.OrderID, ctx.Time().Unix())

	res := coinPayCommRes{
		ID:           pLog.OrderID,
		Address:      string(v.GetStringBytes(protocolType)),
		Amount:       usdtAmount,
		ProtocolType: protocolType,
	}

	helper.Print(ctx, true, res)
}

// DepositCallBack 存款回调
func DepositCallBack(ctx *fasthttp.RequestCtx, p Payment) {

	var (
		err  error
		data paymentCallbackResp
	)
	pLog := &paymentTDLog{
		Merchant:   p.Name(),
		Flag:       "deposit callback",
		Lable:      paymentLogTag,
		RequestURL: string(ctx.RequestURI()),
	}

	if string(ctx.Method()) == fasthttp.MethodGet {
		pLog.RequestBody = ctx.QueryArgs().String()
	}

	if string(ctx.Method()) == fasthttp.MethodPost {
		pLog.RequestBody = ctx.PostArgs().String()
	}

	// 记录请求日志
	defer func() {
		if err != nil {
			pLog.Error = err.Error()
		}

		pLog.ResponseBody = string(ctx.Response.Body())
		pLog.ResponseCode = ctx.Response.StatusCode()
		paymentPushLog(pLog)
	}()

	// 获取并校验回调参数
	data, err = p.PayCallBack(ctx)
	if err != nil {
		ctx.SetBody([]byte(`failed`))
		return
	}

	// 查询订单
	order, err := depositFind(data.OrderID)
	if err != nil {
		err = fmt.Errorf("query order error: [%v]", err)
		ctx.SetBody([]byte(`failed`))
		return
	}

	pLog.Username = order.Username
	pLog.OrderID = data.OrderID

	ch := paymentChannelMatch(order.ChannelID)
	pLog.Channel = string(ch)

	if order.State == DepositSuccess || order.State == DepositCancelled {
		err = fmt.Errorf("duplicated deposite notify: [%d]", order.State)
		ctx.SetBody([]byte(`failed`))
		return
	}

	// usdt 验证usdt金额
	if order.PID == "101003754213878523" {

		hashID := string(ctx.QueryArgs().Peek("hash"))
		// 记录实际入账金额usdt 和 订单hash
		err = depositUpdateUsdtAmount(order.ID, data.Amount, hashID, order.Rate)
		if err != nil {
			ctx.SetBody([]byte(`failed`))
			return
		}

	} else { // 校验money 非usdt渠道需要验证订单金额是否一致

		// 兼容越南盾的单位K 与 人民币元
		if data.Cent == 0 {
			data.Cent = 1000
		}

		orderAmount := fmt.Sprintf("%.4f", order.Amount)
		err = compareAmount(data.Amount, orderAmount, data.Cent)
		if err != nil {
			err = fmt.Errorf("compare amount error: [err: %v, req: %s, origin: %s]", err, data.Amount, orderAmount)
			ctx.SetBody([]byte(`failed`))
			return
		}
	}

	// 修改订单状态
	err = depositUpdate(data.State, order)
	if err != nil {
		err = fmt.Errorf("set order state error: [%v], old state=%d, new state=%d", err, order.State, data.State)
		ctx.SetBody([]byte(`failed`))
		return
	}

	if data.Resp != nil {
		ctx.SetStatusCode(200)
		ctx.SetContentType("application/json")
		bytes, err := jettison.Marshal(data.Resp)
		if err != nil {
			ctx.SetBody([]byte(err.Error()))
			return
		}
		ctx.SetBody(bytes)
		return
	}

	ctx.SetBody([]byte(`success`))
}

// Manual 调用与pid对应的渠道, 发起充值(代付)请求
func Manual(ctx *fasthttp.RequestCtx, pid, amount, bankcardID, bankCode string, user Member) {

	var err error
	pLog := &paymentTDLog{
		Lable:    paymentLogTag,
		Flag:     "deposit",
		Username: user.Username,
	}
	// 记录请求日志
	defer func() {
		if err != nil {
			pLog.Error = fmt.Sprintf("{req: %s, err: %s}", ctx.PostArgs().String(), err.Error())
		}
		paymentPushLog(pLog)
	}()

	p, err := CachePayment(pid)
	if err != nil {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	ch := paymentChannelMatch(p.ChannelID)
	pLog.Merchant = "线下转卡"
	pLog.Channel = string(ch)

	// 检查存款金额是否符合范围
	a, ok := validator.CheckFloatScope(amount, p.Fmin, p.Fmax)
	if !ok {
		helper.Print(ctx, false, helper.AmountOutRange)
		return
	}

	// 检查用户的存款行为是否过于频繁
	err = cacheDepositProcessing(user.UID, time.Now().Unix())
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	// 获取银行卡
	card, err := BankCards(bankcardID)
	if err != nil {
		helper.Print(ctx, false, helper.ChannelBusyTryOthers)
		return
	}

	// 获取附言码
	code, err := DepositManualRemark(bankcardID)
	if err != nil {
		helper.Print(ctx, false, helper.ChannelBusyTryOthers)
		return
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
		"created_at":    ctx.Time().Unix(),
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
		pLog.Error = fmt.Sprintf("insert into table error: [%v]", err)
		helper.Print(ctx, false, helper.DBErr)
		return
	}

	// 记录存款行为
	_ = cacheDepositProcessingInsert(user.UID, pLog.OrderID, ctx.Time().Unix())

	res := offlineCommRes{
		ID:           pLog.OrderID,
		Name:         card.Name,
		CardNo:       card.CardNo,
		RealName:     card.RealName,
		BankAddr:     card.BankAddr,
		ManualRemark: code,
	}

	helper.Print(ctx, true, res)
}

// USDT 线下USDT支付
func USDT(ctx *fasthttp.RequestCtx, pid, amount, addr, protocolType, hashID string, user Member) {

	var err error
	pLog := &paymentTDLog{
		Lable:    paymentLogTag,
		Flag:     "deposit",
		Username: user.Username,
	}
	// 记录请求日志
	defer func() {
		if err != nil {
			pLog.Error = fmt.Sprintf("{req: %s, err: %s, user: %s}", ctx.PostArgs().String(), err.Error(), user.Username)
		}
		paymentPushLog(pLog)
	}()

	p, err := CachePayment(pid)
	if err != nil {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	ch := paymentChannelMatch(p.ChannelID)
	pLog.Merchant = "线下USDT"
	pLog.Channel = string(ch)

	rate, err := USDTConfig()
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	dm, err := decimal.NewFromString(amount)
	if err != nil {
		helper.Print(ctx, false, helper.AmountErr)
		return
	}

	// 发起的usdt金额
	usdtAmount := dm.Mul(decimal.NewFromInt(1000)).DivRound(rate, 3).String()

	// 生成我方存款订单号
	pLog.OrderID = helper.GenId()

	// 检查用户的存款行为是否过于频繁
	err = cacheDepositProcessing(user.UID, time.Now().Unix())
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	d := g.Record{
		"id":                pLog.OrderID,
		"prefix":            meta.Prefix,
		"oid":               pLog.OrderID,
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
		"created_at":        ctx.Time().Unix(),
		"created_uid":       "0",
		"created_name":      "",
		"confirm_at":        "0",
		"confirm_uid":       "0",
		"confirm_name":      "",
		"review_remark":     "",
		"protocol_type":     protocolType,
		"address":           addr,
		"usdt_apply_amount": usdtAmount,
		"rate":              rate.String(),
		"hash_id":           hashID,
		"flag":              DepositFlagUSDT,
		"level":             user.Level,
	}

	// 请求成功插入订单
	err = deposit(d)
	if err != nil {
		pLog.Error = fmt.Sprintf("insert into table error: [%v]", err)
		helper.Print(ctx, false, helper.DBErr)
		return
	}

	// 记录存款行为
	_ = cacheDepositProcessingInsert(user.UID, pLog.OrderID, ctx.Time().Unix())

	res := payCommRes{
		ID: pLog.OrderID,
	}

	helper.Print(ctx, true, res)
}
