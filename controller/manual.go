package controller

import (
	"finance/contrib/helper"
	"finance/model"
	"fmt"

	"github.com/valyala/fasthttp"
)

type ManualController struct{}

// Manual 发起线下转卡
func (that *ManualController) Pay(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	amount := string(ctx.PostArgs().Peek("amount"))
	bid := string(ctx.PostArgs().Peek("bankcard_id"))
	bankCode := string(ctx.PostArgs().Peek("bank_code"))
	fmt.Println("id:", id)
	fmt.Println("Manual: ", string(ctx.PostBody()))

	if id != "767158011957916898" {
		helper.Print(ctx, false, helper.ChannelIDErr)
		return
	}

	if !helper.CtypeDigit(amount) {
		helper.Print(ctx, false, helper.AmountErr)
		return
	}

	if bid == "" || bankCode == "" {
		helper.Print(ctx, false, helper.BankcardIDErr)
		return
	}

	res, err := model.ManualPay(ctx, id, amount, bid, bankCode)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, false, res)
}
