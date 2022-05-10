package controller

import (
	"finance/contrib/helper"
	"finance/model"

	"github.com/valyala/fasthttp"
)

type UsdtController struct{}

func (that *UsdtController) Info(ctx *fasthttp.RequestCtx) {

	res, err := model.UsdtInfo()
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, res)
}

func (that *UsdtController) Update(ctx *fasthttp.RequestCtx) {

	field := string(ctx.PostArgs().Peek("field"))
	value := string(ctx.PostArgs().Peek("value"))
	code := string(ctx.PostArgs().Peek("code"))

	/*
		fmt.Println("field = ", field)
		fmt.Println("value = ", value)
	*/
	if field != "usdt_rate" && field != "usdt_trc_addr" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if !helper.CtypeDigit(code) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	err := model.UsdtUpdate(field, value)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// USDT 发起线下USDT
func (that *UsdtController) Pay(ctx *fasthttp.RequestCtx) {

	amount := string(ctx.PostArgs().Peek("amount"))
	id := string(ctx.PostArgs().Peek("id"))
	addr := string(ctx.PostArgs().Peek("addr"))
	protocolType := string(ctx.PostArgs().Peek("protocol_type"))
	hashID := string(ctx.PostArgs().Peek("hash_id"))

	if protocolType != "TRC20" || addr == "" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if len(hashID) != 64 {
		helper.Print(ctx, false, helper.InvalidTransactionHash)
		return
	}

	if id != "387901070217440117" {
		helper.Print(ctx, false, helper.ChannelIDErr)
		return
	}

	if !helper.CtypeDigit(amount) {
		helper.Print(ctx, false, helper.AmountErr)
		return
	}

	if addr == "" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	model.UsdtPay(ctx, id, amount, addr, protocolType, hashID)
}
