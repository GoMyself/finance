package controller

import (
	"fmt"
	"strings"

	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"

	"github.com/valyala/fasthttp"
)

type ChannelController struct{}

type channelParam struct {
	CateID    string `rule:"digit" msg:"cate_id error" name:"cate_id"`                         // 渠道id
	ChannelID string `rule:"digit" min:"1" max:"100" msg:"channel_id error" name:"channel_id"` // 通道id
	FMin      string `rule:"float" msg:"fmin error" name:"fmin"`                               // 最小支付金额
	FMax      string `rule:"float" msg:"fmax error" name:"fmax"`                               // 最大支付金额
	//St         string `rule:"time" msg:"st error" name:"st"`                                    // 开始时间
	//Et         string `rule:"time" msg:"et error" name:"et"`                                    // 结束时间
	Device     string `rule:"none" msg:"device error" name:"device"`               // 设备号多选逗号分开
	Sort       string `rule:"digit" min:"1" max:"99" msg:"sort error" name:"sort"` // 排序
	Comment    string `rule:"none" msg:"comment error" name:"comment"`             // 备注
	Code       string `rule:"digit" msg:"code error" name:"code"`                  // 动态验证码
	AmountList string `rule:"none" msg:"amount_list error" name:"amount_list"`     // 固定金额列表
}

type updateChannelParam struct {
	ID         string `rule:"digit" msg:"id error" name:"id"`
	FMin       string `rule:"float" msg:"fmin error" name:"fmin"`                  // 最小支付金额
	FMax       string `rule:"float" msg:"fmax error" name:"fmax"`                  // 最大支付金额
	St         string `rule:"time" msg:"st error" name:"st"`                       // 开始时间
	Et         string `rule:"time" msg:"et error" name:"et"`                       // 结束时间
	Device     string `rule:"none" msg:"device error" name:"device"`               // 设备号多选逗号分开
	Sort       string `rule:"digit" min:"1" max:"99" msg:"sort error" name:"sort"` // 排序
	Comment    string `rule:"none" msg:"comment error" name:"comment"`             // 备注
	Code       string `rule:"digit" msg:"code error" name:"code"`                  // 动态验证码
	AmountList string `rule:"none" msg:"amount_list error" name:"amount_list"`     // 固定金额列表
}

type channelListParam struct {
	CateID    string `rule:"digit" default:"0" msg:"cate_id error" name:"cate_id"`       // 渠道id
	ChannelID string `rule:"digit" default:"0" msg:"channel_id error" name:"channel_id"` // 通道id
	Device    string `rule:"none" msg:"device error" name:"device"`                      // 支持设备
	Page      string `rule:"none" name:"page"`
}

type chanStateParam struct {
	ID    string `rule:"digit" default:"0" msg:"id error" name:"id"`
	State string `rule:"digit" min:"0" max:"1" msg:"state error" name:"state"` // 0:关闭1:开启
	Code  string `rule:"digit" msg:"code error" name:"code"`                   // 动态验证码
}

// List 财务管理-渠道管理-通道管理-列表
func (that *ChannelController) List(ctx *fasthttp.RequestCtx) {

	param := channelListParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	var device []string
	if param.Device != "" {
		for _, v := range strings.Split(param.Device, ",") {
			if !validator.CtypeDigit(v) || !validator.CheckIntScope(v, 24, 31) {
				helper.Print(ctx, false, helper.DeviceErr)
				return
			}

			device = append(device, v)
		}
	}

	data, err := model.ChannelList(param.CateID, param.ChannelID, device, param.Page)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

/*
func (that *ChannelController) Cache(ctx *fasthttp.RequestCtx) {
	data := model.ChannelListRedis()
	helper.PrintJson(ctx, true, data)
}
*/

// Insert 财务管理-渠道管理-通道管理-新增
func (that *ChannelController) Insert(ctx *fasthttp.RequestCtx) {

	param := channelParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if param.Comment != "" {
		if !validator.CheckStringLength(param.Comment, 0, 50) {
			helper.Print(ctx, false, helper.RemarkFMTErr)
			return
		}
	}

	if param.AmountList != "" {
		if !validator.CheckStringCommaDigit(param.AmountList) {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}
	}

	var device []string
	if param.Device != "" {
		for _, v := range strings.Split(param.Device, ",") {
			if !validator.CtypeDigit(v) || !validator.CheckIntScope(v, 24, 36) {
				helper.Print(ctx, false, helper.DeviceTypeErr)
				return
			}
			device = append(device, v)
		}
	}

	// 三方渠道
	cate, err := model.CateByID(param.CateID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(cate.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	// 三方通道
	channel, err := model.TunnelByID(param.ChannelID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(channel.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	//content := fmt.Sprintf("新增【渠道名称: %s；通道名称: %s；最小充值金额: %s；最大充值金额: %s】", cate.Name, channel.Name, param.FMin, param.FMax)
	//defer model.SystemLogWrite(content, ctx)

	fields := map[string]string{
		"id":          helper.GenLongId(),
		"cate_id":     param.CateID,
		"channel_id":  param.ChannelID,
		"fmin":        param.FMin,
		"fmax":        param.FMax,
		"st":          "00:00:00",
		"et":          "00:00:00",
		"created_at":  fmt.Sprintf("%d", ctx.Time().Unix()),
		"sort":        param.Sort,
		"comment":     param.Comment,
		"amount_list": param.AmountList,
	}
	err = model.ChannelInsert(fields, device)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// Update 财务管理-渠道管理-通道管理-修改
func (that *ChannelController) Update(ctx *fasthttp.RequestCtx) {

	param := updateChannelParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if param.Comment != "" {
		if !validator.CheckStringLength(param.Comment, 0, 50) {
			helper.Print(ctx, false, helper.RemarkFMTErr)
			return
		}
	}

	var device []string
	if param.Device != "" {
		for _, v := range strings.Split(param.Device, ",") {
			if !validator.CtypeDigit(v) || !validator.CheckIntScope(v, 24, 31) {
				helper.Print(ctx, false, helper.DeviceTypeErr)
				return
			}

			device = append(device, v)
		}
	}

	if param.AmountList != "" {
		if !validator.CheckStringCommaDigit(param.AmountList) {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}
	}

	// 校验渠道id和通道id是否存在
	payment, err := model.ChanExistsByID(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(payment.ID) == 0 {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	if payment.State == "1" {
		helper.Print(ctx, false, helper.UpdateMustCloseFirst)
		return
	}

	// 三方渠道
	cate, err := model.CateByID(payment.CateID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(cate.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	// 三方通道
	channel, err := model.TunnelByID(payment.ChannelID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(channel.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	//content := fmt.Sprintf("编辑【渠道名称: %s；通道名称: %s；最小充值金额: %s；最大充值金额: %s】", cate.Name, channel.Name, param.FMin, param.FMax)
	//defer model.SystemLogWrite(content, ctx)

	fields := map[string]string{
		"id":          param.ID,
		"quota":       "0",
		"gateway":     "",
		"fmin":        param.FMin,
		"fmax":        param.FMax,
		"st":          param.St,
		"et":          param.Et,
		"sort":        param.Sort,
		"comment":     param.Comment,
		"amount_list": param.AmountList,
	}
	err = model.ChannelUpdate(fields, device)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// 财务管理-渠道管理-通道管理-删除
func (that *ChannelController) Delete(ctx *fasthttp.RequestCtx) {

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

	err := model.ChannelDelete(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// UpdateState 财务管理-渠道管理-通道管理-启用/停用
func (that *ChannelController) UpdateState(ctx *fasthttp.RequestCtx) {

	param := chanStateParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	// 校验渠道id和通道id是否存在
	payment, err := model.ChanExistsByID(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(payment.ID) == 0 {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	if payment.State == param.State {
		helper.Print(ctx, false, helper.NoDataUpdate)
		return
	}

	// 三方渠道
	cate, err := model.CateByID(payment.CateID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(cate.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	// 上级渠道关闭的时候不能开启
	if param.State == "1" && cate.State == "0" {
		helper.Print(ctx, false, helper.ParentChannelClosed)
		return

	}

	// 三方通道
	channel, err := model.TunnelByID(payment.ChannelID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(channel.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	/*
		keyword := "开启"
		if param.State == "0" {
			keyword = "关闭"
		}

		content := fmt.Sprintf("%s【渠道名称: %s ；通道名称: %s】", keyword, cate.Name, channel.Name)
		defer model.SystemLogWrite(content, ctx)
	*/

	err = model.ChannelSet(param.ID, param.State)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}
