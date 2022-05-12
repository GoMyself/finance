package controller

import (
	"finance/contrib/helper"
	"finance/model"

	"github.com/valyala/fasthttp"
)

type ManualController struct{}

// Manual 发起线下转卡
func (that *ManualController) Pay(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	amount := string(ctx.PostArgs().Peek("amount"))

	//fmt.Println("id:", id)
	//fmt.Println("Manual: ", string(ctx.PostBody()))

	if !helper.CtypeDigit(id) {
		helper.Print(ctx, false, helper.AmountErr)
		return
	}

	if !helper.CtypeDigit(amount) {
		helper.Print(ctx, false, helper.AmountErr)
		return
	}

	res, err := model.ManualPay(ctx, "766870294997073616", amount)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, res)
}
