package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"

	g "github.com/doug-martin/goqu/v9"
	"github.com/valyala/fasthttp"
	"github.com/wI2L/jettison"
)

// NewestPay 调用与pid对应的渠道, 发起充值(代付)请求
func NewestPay(fctx *fasthttp.RequestCtx, pid, amount, bid string) (map[string]string, error) {

	res := map[string]string{}
	var data paymentDepositResp

	user, err := MemberCache(fctx)
	if err != nil {
		return res, err
	}

	p, err := CachePayment(pid)
	if err != nil {
		return res, errors.New(helper.ChannelNotExist)
	}

	fmt.Println("NewestPay p:", p)
	data, err = Pay(user, p, amount, bid)
	if err != nil {
		/*
			// 兼容日志写入 如果输出的不是标准提示 则使用标准提示输出 记录原始错误信息
			if _, e := strconv.Atoi(err.Error()); e == nil {
				helper.Print(ctx, false, err.Error())
				return
			}
		*/
		//helper.Print(ctx, false, helper.ChannelBusyTryOthers)
		return res, err
	}

	ts := fctx.Time().In(loc).Unix()
	d := g.Record{
		"id":                data.OrderID,
		"prefix":            meta.Prefix,
		"oid":               data.OrderID,
		"uid":               user.UID,
		"top_uid":           user.TopUid,
		"top_name":          user.TopName,
		"parent_name":       user.ParentName,
		"parent_uid":        user.ParentUid,
		"username":          user.Username,
		"channel_id":        p.ChannelID,
		"level":             user.Level,
		"cid":               p.CateID,
		"pid":               p.ID,
		"amount":            amount,
		"usdt_apply_amount": 0,
		"rate":              1,
		"state":             DepositConfirming,
		"finance_type":      helper.TransactionDeposit,
		"automatic":         "1",
		"created_at":        fmt.Sprintf("%d", ts),
		"created_uid":       "0",
		"created_name":      "",
		"confirm_at":        "0",
		"confirm_uid":       "0",
		"confirm_name":      "",
		"review_remark":     "",
		"protocol_type":     "",
		"address":           "",
		"flag":              DepositFlagThird,
		"tester":            user.Tester,
	}

	//只针对越南支付，才统计查看银行编码存库到bank_code字段
	var vNPay = map[string]bool{
		"171560943702910226": true, // VN支付 Online
		"439141987451271871": true, // VN支付 Offline
		"440046584965688018": true, // VN支付 MOMO
		"440058675832531078": true, // VN支付 QR Banking
	}

	if len(bid) > 0 && bid != "0" && vNPay[pid] {
		d["bank_code"] = bid
	}

	fmt.Println("deposit d:", d)
	// 请求成功插入订单
	err = deposit(d)
	if err != nil {
		fmt.Println("insert into table error: = ", err)
		return res, errors.New(helper.DBErr)
	}

	// 记录存款行为
	_ = cacheDepositProcessingInsert(user.UID, data.OrderID, ts)

	res["id"] = data.OrderID
	res["url"] = data.Addr

	return res, nil
}

/*
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
	if protocolType != "TRC20" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	p, err := CachePayment(pid)
	if err != nil {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	usdt_info_temp, err := UsdtInfo()
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	usdt_rate, err := decimal.NewFromString(usdt_info_temp["usdt_rate"])
	if err != nil {
		helper.Print(ctx, false, helper.AmountErr)
		return
	}

	dm, err := decimal.NewFromString(amount)
	if err != nil {
		helper.Print(ctx, false, helper.AmountErr)
		return
	}

	// 提交usdt金额给三方
	usdtAmount := dm.Mul(decimal.NewFromInt(1000)).DivRound(usdt_rate, 3).String()

	payment, ok := paymentRoute[p.CateID]
	if !ok {
		helper.Print(ctx, false, helper.NoPayChannel)
		return
	}

	//ch := paymentChannelMatch(p.ChannelID)

	chChannelTypeById(p.ChannelID)
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
		"rate":              usdt_rate.String(),
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
*/
// DepositCallBack 存款回调
func DepositCallBack(fctx *fasthttp.RequestCtx, payment_id string) {

	var (
		err  error
		data paymentCallbackResp
	)

	p, ok := paymentRoute[payment_id]
	if !ok {
		fmt.Println(payment_id, " not found")
		return
	}

	pLog := paymentTDLog{
		Merchant:   p.Name(),
		Flag:       "1",
		Lable:      paymentLogTag,
		RequestURL: string(fctx.RequestURI()),
	}

	if string(fctx.Method()) == fasthttp.MethodGet {
		pLog.RequestBody = fctx.QueryArgs().String()
	}

	if string(fctx.Method()) == fasthttp.MethodPost {
		pLog.RequestBody = fctx.PostArgs().String()
	}

	// 记录请求日志
	defer func() {
		if err != nil {
			pLog.Error = err.Error()
		}

		pLog.ResponseBody = string(fctx.Response.Body())
		pLog.ResponseCode = fctx.Response.StatusCode()
		paymentPushLog(pLog)
	}()

	// 获取并校验回调参数
	data, err = p.PayCallBack(fctx)
	if err != nil {
		fctx.SetBody([]byte(`failed`))
		return
	}
	pLog.OrderID = data.OrderID

	// 查询订单
	order, err := depositFind(data.OrderID)
	if err != nil {
		err = fmt.Errorf("query order error: [%v]", err)
		fctx.SetBody([]byte(`failed`))
		return
	}

	pLog.Username = order.Username

	ch, err := ChannelTypeById(order.ChannelID)
	if err != nil {
		//return "", errors.New(helper.ChannelNotExist)

		return
	}

	pLog.Channel = ch["name"]

	if order.State == DepositSuccess || order.State == DepositCancelled {
		err = fmt.Errorf("duplicated deposite notify: [%d]", order.State)
		fctx.SetBody([]byte(`failed`))
		return
	}

	// usdt 验证usdt金额
	if order.PID == "101003754213878523" {

		hashID := string(fctx.QueryArgs().Peek("hash"))
		// 记录实际入账金额usdt 和 订单hash
		err = depositUpdateUsdtAmount(order.ID, data.Amount, hashID, order.Rate)
		if err != nil {
			fctx.SetBody([]byte(`failed`))
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
			fctx.SetBody([]byte(`failed`))
			return
		}
	}

	// 修改订单状态
	err = depositUpdate(data.State, order)
	if err != nil {
		err = fmt.Errorf("set order state error: [%v], old state=%d, new state=%d", err, order.State, data.State)
		fctx.SetBody([]byte(`failed`))
		return
	}

	if data.Resp != nil {
		fctx.SetStatusCode(200)
		fctx.SetContentType("application/json")
		bytes, err := jettison.Marshal(data.Resp)
		if err != nil {
			fctx.SetBody([]byte(err.Error()))
			return
		}
		fctx.SetBody(bytes)
		return
	}

	fctx.SetBody([]byte(`success`))
}
