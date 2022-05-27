package controller

import (
	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"
	"github.com/shopspring/decimal"

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
		if !validator.CheckUName(username, 4, 20) {
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

// OfflineToReview 修改订单状态 确认金额
func (that *ManualController) Confirm(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	remark := string(ctx.PostArgs().Peek("remark"))
	amount := ctx.PostArgs().GetUfloatOrZero("amount")

	if !validator.CheckStringDigit(id) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if remark != "" {
		remark = validator.FilterInjection(remark)
	}

	if amount <= 0 {
		helper.Print(ctx, false, helper.AmountErr)
		return
	}

	deposit, err := model.DepositFindOne(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if deposit.State != model.DepositConfirming {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	/*
		// 写入系统日志
		logMsg := fmt.Sprintf("线下转卡【订单id:%s；到账金额:%.4f】", id, amount)
		defer model.SystemLogWrite(logMsg, ctx)
	*/

	rec := g.Record{
		"amount":        amount,
		"review_remark": remark,
		"state":         model.DepositReviewing,
	}

	err = model.DepositRecordUpdate(id, rec)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

//OfflineReview 线下转卡-审核
func (that *ManualController) Review(ctx *fasthttp.RequestCtx) {

	remark := string(ctx.PostArgs().Peek("remark"))
	state := ctx.PostArgs().GetUintOrZero("state")
	id := string(ctx.PostArgs().Peek("id"))

	if remark == "" || !validator.CheckStringDigit(id) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if state != model.DepositSuccess && state != model.DepositCancelled {
		helper.Print(ctx, false, helper.StateParamErr)
		return
	}

	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	deposit, err := model.DepositFindOne(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	/*
		keyword := "通过"
		if state == model.DepositCancelled {
			keyword = "拒绝"
		}

		// 写入系统日志
		logMsg := fmt.Sprintf("线下转卡:%s【订单号: %s；会员账号: %s；金额: %.4f；审核时间: %s】",
			keyword, id, deposit.Username, deposit.Amount, model.TimeFormat(ctx.Time().Unix()))
		defer model.SystemLogWrite(logMsg, ctx)
	*/
	bk, err := model.BankCardByID(deposit.BankcardID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	//可以充值超过当日最大收款限额
	fishAmount, _ := decimal.NewFromString(bk.DailyFinishAmount)
	maxAmount, _ := decimal.NewFromString(bk.DailyMaxAmount)
	totalAmount, _ := decimal.NewFromString(bk.TotalFinishAmount)
	totalMaxAmount, _ := decimal.NewFromString(bk.TotalMaxAmount)

	if state == model.DepositSuccess && (fishAmount.Cmp(maxAmount) >= 0 || fishAmount.Add(decimal.NewFromFloat(deposit.Amount)).GreaterThan(maxAmount) ||
		totalAmount.Add(decimal.NewFromFloat(deposit.Amount)).GreaterThan(totalMaxAmount)) {
		helper.Print(ctx, false, helper.ChangeDepositLimitBeforeActive)
		return
	}

	err = model.ManualReview(id, remark, admin["name"], admin["id"], state, deposit)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	//if state == model.DepositSuccess {
	//	record := g.Record{
	//		"daily_finish_amount": fishAmount.Add(decimal.NewFromFloat(deposit.Amount)).StringFixed(4),
	//		"total_finish_amount": totalAmount.Add(decimal.NewFromFloat(deposit.Amount)).StringFixed(4),
	//	}
	//	model.BankCardUpdate(bk.Id, record)
	//}

	helper.Print(ctx, true, helper.Success)
}
