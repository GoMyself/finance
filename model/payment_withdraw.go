package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"
	"github.com/wI2L/jettison"

	g "github.com/doug-martin/goqu/v9"
	"github.com/valyala/fasthttp"
)

// Withdrawal 提现
func Withdrawal(p Payment, arg WithdrawAutoParam) (string, error) {

	/*
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
	*/
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

	data, err := p.Withdraw(arg)
	if err != nil {
		return "", errors.New(helper.ChannelBusyTryOthers)
	}

	return data.OrderID, nil
}

// WithdrawalCallBack 提款回调
func WithdrawalCallBack(fctx *fasthttp.RequestCtx, payment_id string) {

	var (
		err  error
		data paymentCallbackResp
	)

	p, ok := paymentRoute[payment_id]
	if !ok {
		fmt.Println(payment_id, " not found")
		return
	}

	/*
		pLog := &paymentTDLog{
			Merchant:   p.Name(),
			Flag:       "withdraw callback",
			RequestURL: string(ctx.RequestURI()),
		}

		if string(fctx.Method()) == fasthttp.MethodGet {
			pLog.RequestBody = ctx.QueryArgs().String()
		}

		if string(fctx.Method()) == fasthttp.MethodPost {
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
	*/
	// 获取并校验回调参数
	data, err = p.WithdrawCallBack(fctx)
	if err != nil {
		fctx.SetBody([]byte(`failed`))
		pushLog(err, helper.WithdrawFailure)
		return
	}
	fmt.Println("获取并校验回调参数:", data)

	// 查询订单
	order, err := withdrawFind(data.OrderID)
	if err != nil {
		err = fmt.Errorf("query order error: [%v]", err)
		fctx.SetBody([]byte(`failed`))
		pushLog(err, helper.WithdrawFailure)
		return
	}

	//pLog.Username = order.Username
	//pLog.OrderID = data.OrderID0

	// 提款成功只考虑出款中和代付失败的情况
	// 审核中的状态不用考虑，因为不会走到三方去，出款成功和出款失败是终态也不用考虑
	if order.State != WithdrawDealing && order.State != WithdrawAutoPayFailed {
		err = fmt.Errorf("duplicated Withdrawal notify: [%v]", err)
		fctx.SetBody([]byte(`failed`))
		pushLog(err, helper.WithdrawFailure)
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
			fctx.SetBody([]byte(`failed`))
			pushLog(err, helper.WithdrawFailure)
			return
		}
	}

	// 修改订单状态
	err = withdrawUpdate(data.OrderID, order.UID, order.BID, data.State, fctx.Time())
	if err != nil {
		err = fmt.Errorf("set order state [%d] to [%d] error: [%v]", order.State, data.State, err)
		pushLog(err, helper.WithdrawFailure)
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
