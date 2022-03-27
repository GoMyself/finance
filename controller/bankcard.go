package controller

import (
	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
	"github.com/valyala/fasthttp"
)

type BankCardController struct{}

//Insert 线下卡转卡 添加银行卡
func (that *BankCardController) Insert(ctx *fasthttp.RequestCtx) {

	channelBankID := string(ctx.PostArgs().Peek("id"))
	realName := string(ctx.PostArgs().Peek("real_name"))
	bankAddr := string(ctx.PostArgs().Peek("bank_addr"))
	maxAmount := ctx.PostArgs().GetUfloatOrZero("max_amount")
	cardNo := string(ctx.PostArgs().Peek("card_no"))
	remark := string(ctx.PostArgs().Peek("remark"))

	if bankAddr == "" || maxAmount <= 0 || cardNo == "" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	// 检查该卡号是否已经存在
	_, err := model.BankCardByCol("card_no", cardNo)
	if err == nil {
		helper.Print(ctx, false, helper.BankCardExistErr)
		return
	}

	bank, err := model.ChannelBankByID(channelBankID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	bankCard := model.BankCard{
		ID:            helper.GenId(),
		ChannelBankID: channelBankID,
		Name:          bank.Name,
		CardNo:        cardNo,
		RealName:      realName,
		BankAddr:      bankAddr,
		State:         0,
		Remark:        remark,
		MaxAmount:     maxAmount,
	}

	err = model.BankCardInsert(&bankCard)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	content := fmt.Sprintf("添加银行卡【卡号: %s，最大限额：%.4f, 持卡人姓名：%s】", cardNo, maxAmount, realName)
	defer model.SystemLogWrite(content, ctx)

	helper.Print(ctx, true, helper.Success)
}

//Delete 线下卡专卡 删除银行卡
func (that *BankCardController) Delete(ctx *fasthttp.RequestCtx) {

	id := string(ctx.QueryArgs().Peek("id"))
	if id == "" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	card, err := model.BankCardByID(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	err = model.BankCardDelete(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	// 线下转卡的paymentID  304314961990368154 刷新渠道下银行列表
	_ = model.CacheRefreshPaymentBanks("304314961990368154")

	content := fmt.Sprintf("删除银行卡【卡号: %s，最大限额：%.4f, 持卡人姓名：%s】", card.CardNo, card.MaxAmount, card.RealName)
	defer model.SystemLogWrite(content, ctx)

	helper.Print(ctx, true, helper.Success)
}

//List 银行卡列表
func (that *BankCardController) List(ctx *fasthttp.RequestCtx) {

	cardNo := string(ctx.PostArgs().Peek("card_no"))
	realName := string(ctx.PostArgs().Peek("real_name"))
	channelBankID := string(ctx.PostArgs().Peek("channel_bank_id"))
	page := ctx.PostArgs().GetUintOrZero("page")
	pageSize := ctx.PostArgs().GetUintOrZero("page_size")

	if page == 0 {
		page = 1
	}

	if pageSize == 0 {
		pageSize = 15
	}

	ex := g.Ex{}

	if cardNo != "" {
		ex["card_no"] = cardNo
	}

	if realName != "" {
		ex["real_name"] = realName
	}

	if channelBankID != "" {
		if !validator.CheckStringDigit(channelBankID) {
			helper.Print(ctx, false, helper.ParamErr)
			return
		}
		ex["channel_bank_id"] = channelBankID
	}

	data, err := model.BankCardList(ex, page, pageSize)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

//Update 编辑
func (that *BankCardController) Update(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	state := ctx.PostArgs().GetUintOrZero("state")
	maxAmount := ctx.PostArgs().GetUfloatOrZero("max_amount")
	remark := string(ctx.PostArgs().Peek("remark"))

	bankcard, err := model.BankCardByID(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	// 开启银行卡
	if bankcard.State == 0 && state == 1 {

		amount := bankcard.MaxAmount
		if maxAmount > 0 {
			amount = maxAmount
		}

		err = model.BankCardOpenCondition(bankcard.ID, bankcard.ChannelBankID, amount)
		if err != nil {
			helper.Print(ctx, false, err.Error())
			return
		}
	}

	rec := g.Record{
		"state": state,
	}

	maxAmountLog := ""
	if maxAmount > 0 {
		rec["max_amount"] = maxAmount
		maxAmountLog = fmt.Sprintf(", 最大限额:%.4f-> %.4f,", bankcard.MaxAmount, maxAmount)
	}

	if remark != "" {
		rec["remark"] = remark
	}

	err = model.BankCardUpdate(id, rec)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if bankcard.State != state {
		// 线下转卡的paymentID  304314961990368154 刷新渠道下银行列表
		_ = model.CacheRefreshPaymentBanks("304314961990368154")
	}

	content := fmt.Sprintf("编辑银行卡【卡号: %s, %s 持卡人姓名:%s，状态: %d->%d】",
		bankcard.CardNo, maxAmountLog, bankcard.RealName, bankcard.State, state)
	defer model.SystemLogWrite(content, ctx)

	helper.Print(ctx, true, helper.Success)
}

//BankCards 银行卡列表 前台
func (that *BankCardController) BankCards(ctx *fasthttp.RequestCtx) {

	channelBankID := string(ctx.PostArgs().Peek("channel_bank_id"))
	page := ctx.PostArgs().GetUintOrZero("page")
	pageSize := ctx.PostArgs().GetUintOrZero("page_size")

	if page == 0 {
		page = 1
	}

	if pageSize == 0 {
		pageSize = 15
	}

	ex := g.Ex{
		"state": 1,
	}

	if channelBankID == "" || !validator.CheckStringDigit(channelBankID) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	ex["channel_bank_id"] = channelBankID

	data, err := model.BankCardList(ex, page, pageSize)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}
