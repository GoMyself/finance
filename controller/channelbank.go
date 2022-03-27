package controller

import (
	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"
	"fmt"
	"github.com/valyala/fasthttp"
)

type ChannelBankController struct{}

type chanBankListParam struct {
	CateID    string `rule:"digit" default:"0" msg:"cate_id error" name:"cate_id"`       // 渠道id
	ChannelID string `rule:"digit" default:"0" msg:"channel_id error" name:"channel_id"` // 通道id
	Page      uint16 `rule:"digit" default:"1" min:"1" msg:"page error" name:"page"`
	PageSize  uint16 `rule:"digit" default:"10" min:"10" max:"200" msg:"page_size error" name:"page_size"`
}

type chanBankStateParam struct {
	ID    string `rule:"digit" default:"0" msg:"id error" name:"id"`
	State string `rule:"digit" min:"0" max:"1" msg:"state error" name:"state"` // 0:关闭1:开启
	Code  string `rule:"digit" msg:"code error" name:"code"`                   // 动态验证码
}

// List 财务管理-渠道管理-通道银行管理-列表
func (that *ChannelBankController) List(ctx *fasthttp.RequestCtx) {

	param := chanBankListParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	data, err := model.ChannelBankList(param.CateID, param.ChannelID, param.Page, param.PageSize)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// Insert 财务管理-渠道管理-通道银行管理-新增
func (that *ChannelBankController) Insert(ctx *fasthttp.RequestCtx) {

	cateID := string(ctx.PostArgs().Peek("cate_id")) // 渠道id
	if !validator.CheckStringDigit(cateID) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	channelID := string(ctx.PostArgs().Peek("channel_id")) // 通道id
	if !validator.CheckStringDigit(channelID) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	bankID := string(ctx.PostArgs().Peek("bank_id")) // 银行id
	if !validator.CheckStringDigit(bankID) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	name := string(ctx.PostArgs().Peek("name")) // 银行name
	if name == "" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	sort := string(ctx.PostArgs().Peek("sort")) // 排序
	if !validator.CheckStringDigit(sort) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	bankCode := string(ctx.PostArgs().Peek("bank_code")) // 银行别名
	if bankCode == "" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	cateName, channelName, err := model.TunnelAndChannelGetName(cateID, channelID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	logMsg := fmt.Sprintf("新增【渠道名称: %s；通道名称: %s；银行名称: %s】", cateName, channelName, name)
	defer model.SystemLogWrite(logMsg, ctx)

	fields := map[string]string{
		"id":         helper.GenId(),
		"bank_id":    bankID,
		"name":       name,
		"cate_id":    cateID,
		"channel_id": channelID,
		"code":       bankCode,
		"sort":       sort,
	}
	err = model.ChannelBankInsert(fields)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// Update 财务管理-渠道管理-通道银行管理-修改
func (that *ChannelBankController) Update(ctx *fasthttp.RequestCtx) {

	ID := string(ctx.PostArgs().Peek("id")) // id
	if !validator.CheckStringDigit(ID) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	channelID := string(ctx.PostArgs().Peek("channel_id")) // 通道id
	if !validator.CheckStringDigit(channelID) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	bankID := string(ctx.PostArgs().Peek("bank_id")) // 银行id
	if !validator.CheckStringDigit(bankID) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	name := string(ctx.PostArgs().Peek("name")) // 银行name
	if name == "" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	sort := string(ctx.PostArgs().Peek("sort")) // 排序
	if !validator.CheckStringDigit(sort) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	bankCode := string(ctx.PostArgs().Peek("bank_code")) // 银行别名
	if bankCode == "" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	bank, err := model.ChannelBankByID(ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(bank.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	cateName, channelName, err := model.TunnelAndChannelGetName(bank.CateID, channelID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	logMsg := fmt.Sprintf("编辑【渠道名称: %s；通道名称: %s；银行名称: %s】", cateName, channelName, name)
	defer model.SystemLogWrite(logMsg, ctx)

	fields := map[string]string{
		"id":         ID,
		"bank_id":    bankID,
		"name":       name,
		"code":       bankCode,
		"sort":       sort,
		"payment_id": channelID,
	}
	err = model.ChannelBankUpdate(fields)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// Delete 财务管理-渠道管理-通道银行管理-删除
func (that *ChannelBankController) Delete(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	if !validator.CheckStringDigit(id) {
		helper.Print(ctx, false, helper.IDErr)
		return
	}

	code := string(ctx.PostArgs().Peek("code"))
	if !validator.CheckStringDigit(code) {
		helper.Print(ctx, false, helper.CodeErr)
		return
	}

	err := model.ChannelBankDelete(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// UpdateState 财务管理-渠道管理-通道银行管理-启用/停用
func (that *ChannelBankController) UpdateState(ctx *fasthttp.RequestCtx) {

	param := chanBankStateParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	bank, err := model.ChannelBankByID(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(bank.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	if bank.State == param.State {
		helper.Print(ctx, false, helper.NoDataUpdate)
		return
	}

	channelIDMap, err := model.PaymentIDMapToChanID([]string{bank.PaymentID})
	if err != nil {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	if _, ok := channelIDMap[bank.PaymentID]; !ok {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	cateName, channelName, err := model.TunnelAndChannelGetName(bank.CateID, channelIDMap[bank.PaymentID])
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	keyword := "开启"
	if param.State == "0" {
		keyword = "关闭"
	}

	logMsg := fmt.Sprintf("%s【渠道名称: %s；通道名称: %s；银行名称: %s】", keyword, cateName, channelName, bank.Name)
	defer model.SystemLogWrite(logMsg, ctx)

	err = model.ChannelBankSet(param.ID, param.State, bank.PaymentID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}
