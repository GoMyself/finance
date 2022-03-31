package controller

import (
	"finance/model"

	"github.com/valyala/fasthttp"
)

//CallBackController 支付代付回调Controller
type CallBackController struct{}

//UZD 存款回调
func (that *CallBackController) UZD(ctx *fasthttp.RequestCtx) {

	//model.DepositCallBack(ctx, model.UzPay)
	model.DepositCallBack(ctx, "1")
}

//UZW 提款回调
func (that *CallBackController) UZW(ctx *fasthttp.RequestCtx) {

	//model.WithdrawalCallBack(ctx, model.UzPay)
	model.WithdrawalCallBack(ctx, "1")
}

func (that *CallBackController) WD(ctx *fasthttp.RequestCtx) {

	//model.DepositCallBack(ctx, model.WPay)
	model.DepositCallBack(ctx, "6")
}

func (that *CallBackController) WW(ctx *fasthttp.RequestCtx) {

	//model.WithdrawalCallBack(ctx, model.WPay)
	model.WithdrawalCallBack(ctx, "6")
}

//YFBD YFB 回调
func (that *CallBackController) YFBD(ctx *fasthttp.RequestCtx) {

	//model.DepositCallBack(ctx, model.YFB)
	model.DepositCallBack(ctx, "7")
}

//YFBW 优付宝 提款回调
func (that *CallBackController) YFBW(ctx *fasthttp.RequestCtx) {

	//model.WithdrawalCallBack(ctx, model.YFB)
	model.WithdrawalCallBack(ctx, "7")
}

// FYD 凤扬存款回调
func (that *CallBackController) FYD(ctx *fasthttp.RequestCtx) {

	//model.DepositCallBack(ctx, model.FyPay)
	model.DepositCallBack(ctx, "9")
}

// FYW 凤扬取款回调
func (that *CallBackController) FYW(ctx *fasthttp.RequestCtx) {

	//model.WithdrawalCallBack(ctx, model.FyPay)
	model.WithdrawalCallBack(ctx, "9")
}

// QuickD quick pay存款回调
func (that *CallBackController) QuickD(ctx *fasthttp.RequestCtx) {

	//model.DepositCallBack(ctx, model.QuickPay)
	model.DepositCallBack(ctx, "10")
}

// QuickW quick pay 取款回调
func (that *CallBackController) QuickW(ctx *fasthttp.RequestCtx) {

	//model.WithdrawalCallBack(ctx, model.QuickPay)
	model.WithdrawalCallBack(ctx, "10")
}

// UsdtD 充值成功确认
func (that *CallBackController) UsdtD(ctx *fasthttp.RequestCtx) {

	//model.DepositCallBack(ctx, model.USDTPay)
	model.DepositCallBack(ctx, "11")
}

// YND 越南支付代收回调
func (that *CallBackController) YND(ctx *fasthttp.RequestCtx) {
	//model.DepositCallBack(ctx, model.YNPAY)
	model.DepositCallBack(ctx, "16")
}

// YNW 越南支付代付回调
func (that *CallBackController) YNW(ctx *fasthttp.RequestCtx) {

	//model.WithdrawalCallBack(ctx, model.YNPAY)
	model.WithdrawalCallBack(ctx, "16")
}

func (that *CallBackController) VTD(ctx *fasthttp.RequestCtx) {

	//model.DepositCallBack(ctx, model.WPay)
	model.DepositCallBack(ctx, "6")
}

func (that *CallBackController) VTW(ctx *fasthttp.RequestCtx) {

	//model.WithdrawalCallBack(ctx, model.WPay)
	model.WithdrawalCallBack(ctx, "6")
}
