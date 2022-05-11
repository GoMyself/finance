package controller

import (
	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"
	"fmt"
	"strconv"

	"github.com/valyala/fasthttp"
)

type VipController struct{}

type vipParam struct {
	Vip       int    `rule:"digit" min:"1" msg:"vip error" name:"vip"`                         // 会员等级
	CateID    string `rule:"digit" msg:"cate_id error" name:"cate_id"`                         // 渠道id
	ChannelID string `rule:"digit" min:"1" max:"100" msg:"channel_id error" name:"channel_id"` // 通道id
	FMin      string `rule:"float" min:"0" msg:"fmin error" name:"fmin"`                       // 最小支付金额
	FMax      string `rule:"float" min:"0" msg:"fmax error" name:"fmax"`                       // 最大支付金额
	Code      string `rule:"digit" msg:"code error" name:"code"`                               // 动态验证码
}

type vipUpdateParam struct {
	ID   string `rule:"digit" msg:"id error" name:"id"`
	FMin string `rule:"float" min:"0" msg:"fmin error" name:"fmin"` // 最小支付金额
	FMax string `rule:"float" min:"0" msg:"fmax error" name:"fmax"` // 最大支付金额
	Code string `rule:"digit" msg:"code error" name:"code"`         // 动态验证码
}

type vipStateParam struct {
	ID    string `rule:"digit" default:"0" msg:"id error" name:"id"`
	State string `rule:"digit" min:"0" max:"1" msg:"state error" name:"state"` // 0:关闭1:开启
	Code  string `rule:"digit" msg:"code error" name:"code"`                   // 动态验证码
}

// List 财务管理-渠道管理-会员等级通道-列表
func (that *VipController) List(ctx *fasthttp.RequestCtx) {

	level := string(ctx.QueryArgs().Peek("level"))
	flags := string(ctx.QueryArgs().Peek("flags"))

	if !helper.CtypeDigit(level) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}
	if !helper.CtypeDigit(flags) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	data, err := model.VipList(level, flags)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// Insert 财务管理-渠道管理-会员等级通道-新增
func (that *VipController) Insert(ctx *fasthttp.RequestCtx) {

	param := vipParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	/*
		cateName, channelName, err := model.TunnelAndChannelGetName(param.CateID, param.ChannelID)
		if err != nil {
			helper.Print(ctx, false, err.Error())
			return
		}


			logMsg := fmt.Sprintf("新增【渠道名称: %s；通道名称: %s；会员等级: %d；存款金额最小值: %s；存款金额最大值: %s】",
				cateName, channelName, param.Vip-1, param.FMin, param.FMax)
			defer model.SystemLogWrite(logMsg, ctx)
	*/

	fields := map[string]string{
		"id":         helper.GenId(),
		"cate_id":    param.CateID,
		"channel_id": param.ChannelID,
		"vip":        strconv.Itoa(param.Vip),
		"fmin":       param.FMin,
		"fmax":       param.FMax,
	}

	err = model.VipInsert(fields)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// Update 财务管理-渠道管理-会员等级通道-修改
func (that *VipController) Update(ctx *fasthttp.RequestCtx) {

	param := vipUpdateParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	vip, err := model.VipByID(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(vip.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	if vip.State == "1" {
		helper.Print(ctx, false, helper.UpdateMustCloseFirst)
		return
	}

	channelIDMap, err := model.PaymentIDMapToChanID([]string{vip.PaymentID})
	if err != nil {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	if _, ok := channelIDMap[vip.PaymentID]; !ok {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	/*
		cateName, channelName, err := model.TunnelAndChannelGetName(vip.CateID, channelIDMap[vip.PaymentID])
		if err != nil {
			helper.Print(ctx, false, err.Error())
			return
		}

		level, _ := strconv.Atoi(vip.Vip)


			logMsg := fmt.Sprintf("编辑【渠道名称: %s；通道名称: %s；会员等级: %d；存款金额最小值: %s；存款金额最大值: %s】",
				cateName, channelName, level-1, param.FMin, param.FMax)
			defer model.SystemLogWrite(logMsg, ctx)
	*/
	fields := map[string]string{
		"id":   param.ID,
		"fmin": param.FMin,
		"fmax": param.FMax,
	}

	err = model.VipUpdate(vip.PaymentID, fields)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// Delete 财务管理-渠道管理-会员等级通道-删除
func (that *VipController) Delete(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	if !validator.CheckStringDigit(id) {
		helper.Print(ctx, false, helper.IDErr)
		return
	}

	code := string(ctx.PostArgs().Peek("code"))
	if !validator.CheckStringDigit(code) {
		helper.Print(ctx, false, "code error")
		return
	}

	vip, err := model.VipByID(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(vip.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	if vip.State == "1" {
		helper.Print(ctx, false, helper.DeleteMustCloseFirst)
		return
	}

	channelIDMap, err := model.PaymentIDMapToChanID([]string{vip.PaymentID})
	if err != nil {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	if _, ok := channelIDMap[vip.PaymentID]; !ok {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	/*
		cateName, channelName, err := model.TunnelAndChannelGetName(vip.CateID, channelIDMap[vip.PaymentID])
		if err != nil {
			helper.Print(ctx, false, err.Error())
			return
		}


			level, _ := strconv.Atoi(vip.Vip)
			logMsg := fmt.Sprintf("删除【渠道名称: %s；通道名称: %s；会员等级: %d；存款金额最小值: %s；存款金额最大值: %s】",
				cateName, channelName, level-1, vip.Fmin, vip.Fmax)
			defer model.SystemLogWrite(logMsg, ctx)
	*/
	err = model.VipDelete(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// UpdateState 财务管理-渠道管理-会员等级通道-启用/停用
func (that *VipController) UpdateState(ctx *fasthttp.RequestCtx) {

	param := vipStateParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	vip, err := model.VipByID(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}
	fmt.Println("UpdateState VipByID = ", vip)

	if len(vip.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	if vip.State == param.State {
		helper.Print(ctx, false, helper.NoDataUpdate)
		return
	}

	channelIDMap, err := model.PaymentIDMapToChanID([]string{vip.PaymentID})
	if err != nil {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	if _, ok := channelIDMap[vip.PaymentID]; !ok {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	/*
		cateName, channelName, err := model.TunnelAndChannelGetName(vip.CateID, channelIDMap[vip.PaymentID])
		if err != nil {
			helper.Print(ctx, false, err.Error())
			return
		}

		keyword := "启用"
		if param.State == "0" {
			keyword = "禁用"
		}

		level, _ := strconv.Atoi(vip.Vip)
		logMsg := fmt.Sprintf("%s【渠道名称: %s；通道名称: %s；会员等级: %d；存款金额最小值: %s；存款金额最大值: %s】",
			keyword, cateName, channelName, level-1, vip.Fmin, vip.Fmax)
		defer model.SystemLogWrite(logMsg, ctx)
	*/
	err = model.VipSet(param.ID, param.State, vip)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}
