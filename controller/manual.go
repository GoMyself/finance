package controller

import (
	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"

	g "github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
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

// Offline 线下转卡 入款订单或者审核列表
func (that *ManualController) List(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	minAmount := string(ctx.PostArgs().Peek("min_amount"))
	username := string(ctx.PostArgs().Peek("username"))
	maxAmount := string(ctx.PostArgs().Peek("max_amount"))
	cardNo := string(ctx.PostArgs().Peek("card_no"))
	startTime := string(ctx.PostArgs().Peek("start_time"))
	endTime := string(ctx.PostArgs().Peek("end_time"))
	bankCode := string(ctx.PostArgs().Peek("bank_code"))
	page := ctx.PostArgs().GetUintOrZero("page")
	pageSize := ctx.PostArgs().GetUintOrZero("page_size")
	state := ctx.PostArgs().GetUintOrZero("state")
	flag := ctx.PostArgs().GetUintOrZero("flag")

	if page == 0 || pageSize == 0 {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if flag != model.DepositFlagManual && flag != model.DepositFlagUSDT {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if state != model.DepositReviewing && state != model.DepositConfirming {
		helper.Print(ctx, false, helper.StateParamErr)
		return
	}

	ex := g.Ex{
		"state": state,
		"flag":  flag,
	}
	if username != "" {
		if !validator.CheckUName(username, 4, 9) {
			helper.Print(ctx, false, helper.UsernameErr)
			return
		}
		ex["username"] = username
	}

	if id != "" {
		if !validator.CheckStringDigit(id) {
			helper.Print(ctx, false, helper.IDErr)
			return
		}
		ex["id"] = id
	}

	if minAmount != "" && maxAmount != "" {
		ex["amount"] = g.Op{"between": exp.NewRangeVal(minAmount, maxAmount)}
	}

	if bankCode != "" {
		if !validator.CheckStringDigit(bankCode) {
			helper.Print(ctx, false, helper.ParamErr)
			return
		}
		ex["bank_code"] = bankCode
	}

	if cardNo != "" {
		if !validator.CheckStringDigit(cardNo) {
			helper.Print(ctx, false, helper.ParamErr)
			return
		}
		ex["bank_no"] = cardNo
	}

	data, err := model.ManualList(ex, startTime, endTime, page, pageSize)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}
