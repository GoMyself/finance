package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"

	g "github.com/doug-martin/goqu/v9"
	"github.com/valyala/fasthttp"
)

// Withdrawal 提现
func Withdrawal(p Payment, arg WithdrawAutoParam) (string, error) {

	pLog := &paymentTDLog{
		OrderID: arg.OrderID,
		Channel: "withdraw",
		Flag:    "withdraw",
	}

	defer func() {
		pLog.Merchant = p.Name()
		// 记录日志
		paymentPushLog(pLog)
	}()

	// 记录日志
	pLog.Merchant = p.Name()

	// 维护订单 渠道信息
	ex := g.Ex{
		"id": arg.OrderID,
	}
	record := g.Record{
		"pid": arg.PaymentID,
	}
	err := withdrawUpdateInfo(ex, record)
	if err != nil {
		return "", pushLog(err, helper.DBErr)
	}

	data, err := p.Withdraw(pLog, arg)
	if err != nil {
		return "", errors.New(helper.ChannelBusyTryOthers)
	}

	return data.OrderID, nil
}

// WithdrawalCallBack 提款回调
func WithdrawalCallBack(ctx *fasthttp.RequestCtx, payment_id string) {

	var (
		err  error
		data paymentCallbackResp
	)

	p, ok := paymentRoute[payment_id]
	if !ok {
		fmt.Println(payment_id, " not found")
		return
	}

	pLog := &paymentTDLog{
		Merchant:   p.Name(),
		Flag:       "withdraw callback",
		RequestURL: string(ctx.RequestURI()),
	}

	if string(ctx.Method()) == fasthttp.MethodGet {
		pLog.RequestBody = ctx.QueryArgs().String()
	}

	if string(ctx.Method()) == fasthttp.MethodPost {
		pLog.RequestBody = ctx.PostArgs().String()
	}

	// defer记录请求日志
	defer func() {
		if err != nil {
			pLog.Error = err.Error()
		}

		pLog.ResponseBody = string(ctx.Response.Body())
		pLog.ResponseCode = ctx.Response.StatusCode()
		paymentPushLog(pLog)
	}()

	// 获取并校验回调参数
	data, err = p.WithdrawCallBack(ctx)
	if err != nil {
		ctx.SetBody([]byte(`failed`))
		return
	}

	// 查询订单
	order, err := withdrawFind(data.OrderID)
	if err != nil {
		err = fmt.Errorf("query order error: [%v]", err)
		ctx.SetBody([]byte(`failed`))
		return
	}

	pLog.Username = order.Username
	pLog.OrderID = data.OrderID

	// 提款成功只考虑出款中和代付失败的情况
	// 审核中的状态不用考虑，因为不会走到三方去，出款成功和出款失败是终态也不用考虑
	if order.State != WithdrawDealing && order.State != WithdrawAutoPayFailed {
		err = fmt.Errorf("duplicated Withdrawal notify: [%v]", err)
		ctx.SetBody([]byte(`failed`))
		return
	}

	if data.Amount != "-1" {
		// 校验money, 暂时不处理订单与最初订单不一致的情况
		// 兼容越南盾的单位K 与 人民币元
		if data.Cent == 0 {
			data.Cent = 1000
		}
		err = compareAmount(data.Amount, fmt.Sprintf("%.4f", order.Amount), data.Cent)
		if err != nil {
			err = fmt.Errorf("compare amount error: [%v]", err)
			ctx.SetBody([]byte(`failed`))
			return
		}
	}

	// 修改订单状态
	err = withdrawUpdate(data.OrderID, order.UID, order.BID, data.State, ctx.Time())
	if err != nil {
		err = fmt.Errorf("set order state [%d] to [%d] error: [%v]", order.State, data.State, err)
		ctx.SetBody([]byte(`failed`))
		return
	}

	ctx.SetBody([]byte(`success`))
}
