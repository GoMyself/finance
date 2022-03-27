package controller

import (
	"finance/model"
	"github.com/valyala/fasthttp"
)

//CallBackController 支付代付回调Controller
type CallBackController struct{}

//UZD 存款回调
func (that *CallBackController) UZD(ctx *fasthttp.RequestCtx) {

	model.DepositCallBack(ctx, model.UzPay)
}

//UZW 提款回调
func (that *CallBackController) UZW(ctx *fasthttp.RequestCtx) {

	model.WithdrawalCallBack(ctx, model.UzPay)
}

func (that *CallBackController) WD(ctx *fasthttp.RequestCtx) {

	model.DepositCallBack(ctx, model.WPay)
}

func (that *CallBackController) WW(ctx *fasthttp.RequestCtx) {

	model.WithdrawalCallBack(ctx, model.WPay)
}

//YFBD YFB 回调
func (that *CallBackController) YFBD(ctx *fasthttp.RequestCtx) {

	model.DepositCallBack(ctx, model.YFB)
}

//YFBW 优付宝 提款回调
func (that *CallBackController) YFBW(ctx *fasthttp.RequestCtx) {

	model.WithdrawalCallBack(ctx, model.YFB)
}

// FYD 凤扬存款回调
func (that *CallBackController) FYD(ctx *fasthttp.RequestCtx) {

	model.DepositCallBack(ctx, model.FyPay)
}

// FYW 凤扬取款回调
func (that *CallBackController) FYW(ctx *fasthttp.RequestCtx) {

	model.WithdrawalCallBack(ctx, model.FyPay)
}

// QuickD quick pay存款回调
func (that *CallBackController) QuickD(ctx *fasthttp.RequestCtx) {

	model.DepositCallBack(ctx, model.QuickPay)
}

// QuickW quick pay 取款回调
func (that *CallBackController) QuickW(ctx *fasthttp.RequestCtx) {

	model.WithdrawalCallBack(ctx, model.QuickPay)
}

// UsdtD 充值成功确认
func (that *CallBackController) UsdtD(ctx *fasthttp.RequestCtx) {
	model.DepositCallBack(ctx, model.USDTPay)
}

// YND 越南支付代收回调
func (that *CallBackController) YND(ctx *fasthttp.RequestCtx) {
	model.DepositCallBack(ctx, model.YNPAY)
}

// YNW 越南支付代付回调
func (that *CallBackController) YNW(ctx *fasthttp.RequestCtx) {
	model.WithdrawalCallBack(ctx, model.YNPAY)
}
