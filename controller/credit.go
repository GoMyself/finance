package controller

import (
	"fmt"
	"strings"

	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"

	"github.com/valyala/fasthttp"
)

type CreditLevelController struct{}

type creditListParam struct {
	CreditLevel string `rule:"digit" default:"0" min:"0" max:"100" msg:"credit_level error" name:"credit_level"` // 信用等级
	ChanName    string `rule:"none"  msg:"cate_name error" name:"cate_name"`                                     // 渠道名称
	Page        uint16 `rule:"digit" default:"1" min:"1" msg:"page error" name:"page"`
	PageSize    uint16 `rule:"digit" default:"10" min:"10" max:"200" msg:"page_size error" name:"page_size"`
}

type creditParam struct {
	CreditLevel string `rule:"digit" min:"1" max:"100" msg:"credit_level error" name:"credit_level"` // 信用等级
	CateID      string `rule:"digit" msg:"cate_id error" name:"cate_id"`                             // 渠道id
	ChannelID   string `rule:"digit" min:"1" max:"100" msg:"channel_id error" name:"channel_id"`     // 通道id
	FMin        string `rule:"float" min:"0" msg:"fmin error" name:"fmin"`                           // 最小支付金额
	FMax        string `rule:"float" min:"0" msg:"fmax error" name:"fmax"`                           // 最大支付金额
}

type creditUpdateParam struct {
	ID   string `rule:"digit" msg:"id error" name:"id"`
	FMin string `rule:"float" min:"0" msg:"fmin error" name:"fmin"` // 最小支付金额
	FMax string `rule:"float" min:"0" msg:"fmax error" name:"fmax"` // 最大支付金额
}

type creditStateParam struct {
	ID    string `rule:"digit" default:"0" msg:"id error" name:"id"`
	State string `rule:"digit" min:"0" max:"1" msg:"state error" name:"state"` // 0:关闭1:开启
}

type memberCreditListParam struct {
	CreditLevelID string `rule:"digit" min:"1" msg:"credit_level_id error" name:"credit_level_id"` // 信用等级id
	Users         string `rule:"none"  msg:"users error" name:"users"`                             // 会员账号
	Page          uint16 `rule:"digit" default:"1" min:"1" msg:"page error" name:"page"`
	PageSize      uint16 `rule:"digit" default:"10" min:"10" max:"200" msg:"page_size error" name:"page_size"`
}

type memberCreditParam struct {
	CreditLevelID string `rule:"digit" min:"1" msg:"credit_level_id error" name:"credit_level_id"` // 信用等级id
	Users         string `rule:"none"  msg:"users error" name:"users"`                             // 会员账号
}

// Insert 财务管理-渠道管理-会员信用等级-新增
func (that *CreditLevelController) Insert(ctx *fasthttp.RequestCtx) {

	param := creditParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	/*
		cateName, channelName, err := model.TunnelAndChannelGetName(param.CateID, param.ChannelID)
		if err != nil {
			helper.Print(ctx, false, err.Error())
			return
		}

		// 写入系统日志
		logMsg := fmt.Sprintf("新增【信用等级: %s；渠道名称: %s；通道名称: %s；存款金额最小值: %s；存款金额最大值: %s】",
			param.CreditLevel, cateName, channelName, param.FMin, param.FMax)
		defer model.SystemLogWrite(logMsg, ctx)
	*/

	fields := map[string]string{
		"id":         helper.GenId(),
		"level":      param.CreditLevel,
		"cate_id":    param.CateID,
		"channel_id": param.ChannelID,
		"fmin":       param.FMin,
		"fmax":       param.FMax,
		"created_at": fmt.Sprintf("%d", ctx.Time().Unix()),
	}
	err = model.CreditLevelInsert(fields)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// Update 财务管理-渠道管理-会员信用等级-修改
func (that *CreditLevelController) Update(ctx *fasthttp.RequestCtx) {

	param := creditUpdateParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	credit, err := model.CreditLevelByID(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(credit.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	if credit.State == "1" {
		helper.Print(ctx, false, helper.UpdateMustCloseFirst)
		return
	}

	/*
		cateName, channelName, err := model.TunnelAndChannelGetName(credit.CateID, credit.ChannelID)
		if err != nil {
			helper.Print(ctx, false, err.Error())
			return
		}

		// 写入系统日志
		logMsg := fmt.Sprintf("编辑【信用等级: %d；渠道名称: %s；通道名称: %s；存款金额最小值: %s；存款金额最大值: %s】",
			credit.Level, cateName, channelName, param.FMin, param.FMax)
		defer model.SystemLogWrite(logMsg, ctx)
	*/
	fields := map[string]string{
		"id":   param.ID,
		"fmin": param.FMin,
		"fmax": param.FMax,
	}
	err = model.CreditLevelUpdate(credit.PaymentID, fields)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// List 财务管理-渠道管理-会员信用等级-列表
func (that *CreditLevelController) List(ctx *fasthttp.RequestCtx) {

	param := creditListParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if param.ChanName != "" {
		if !validator.CheckStringCHNAlnum(param.ChanName) ||
			!validator.CheckStringLength(param.ChanName, 1, 20) {
			helper.Print(ctx, false, helper.CateNameErr)
			return
		}
	}

	data, err := model.CreditLevelList(param.CreditLevel, param.ChanName, param.Page, param.PageSize)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// UpdateState 财务管理-渠道管理-会员信用等级-启用/停用
func (that *CreditLevelController) UpdateState(ctx *fasthttp.RequestCtx) {

	param := creditStateParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	credit, err := model.CreditLevelByID(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(credit.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	if credit.State == param.State {
		helper.Print(ctx, false, helper.NoDataUpdate)
		return
	}

	/*
		cateName, channelName, err := model.TunnelAndChannelGetName(credit.CateID, credit.ChannelID)
		if err != nil {
			helper.Print(ctx, false, err.Error())
			return
		}


			keyword := "启用"
			if param.State == "0" {
				keyword = "禁用"
			}
			// 写入系统日志
			logMsg := fmt.Sprintf("%s【信用等级: %d；渠道名称: %s；通道名称: %s；存款金额最小值: %d；存款金额最大值: %d】",
				keyword, credit.Level, cateName, channelName, credit.Fmin, credit.Fmax)
			defer model.SystemLogWrite(logMsg, ctx)
	*/

	err = model.CreditLevelUpdateState(param.ID, param.State, credit.PaymentID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// MemberInsert 财务管理-渠道管理-会员信用等级-新增会员
func (that *CreditLevelController) MemberInsert(ctx *fasthttp.RequestCtx) {

	param := memberCreditParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	var users []string
	for _, v := range strings.Split(param.Users, ",") {
		if !validator.CheckUName(v, 5, 14) {
			helper.Print(ctx, false, helper.UsernameErr)
			return
		}
		users = append(users, v)
	}
	if len(users) == 0 || len(users) > 10 {
		helper.Print(ctx, false, helper.UsernameErr)
		return
	}

	creditLevel, err := model.CreditLevelByID(param.CreditLevelID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(creditLevel.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	/*
		cateName, channelName, err := model.TunnelAndChannelGetName(creditLevel.CateID, creditLevel.ChannelID)
		if err != nil {
			helper.Print(ctx, false, err.Error())
			return
		}

		logMsg := fmt.Sprintf("新增【会员账号: %s；信用等级: %d；渠道名称: %s；通道名称: %s】",
			param.Users, creditLevel.Level, cateName, channelName)
		defer model.SystemLogWrite(logMsg, ctx)
	*/

	err = model.MemberCreditLevelInsert(param.CreditLevelID, ctx.Time(), users)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// MemberList 财务管理-渠道管理-会员信用等级-列表会员
func (that *CreditLevelController) MemberList(ctx *fasthttp.RequestCtx) {

	param := memberCreditListParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	var users []string
	if param.Users != "" {
		for _, v := range strings.Split(param.Users, ",") {
			if !validator.CheckUName(v, 5, 14) {
				helper.Print(ctx, false, helper.UsernameErr)
				return
			}
			users = append(users, v)
		}
	}

	data, err := model.MemberCreditLevelList(param.CreditLevelID, param.Page, param.PageSize, users)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// MemberDelete 财务管理-渠道管理-会员信用等级-删除会员
func (that *CreditLevelController) MemberDelete(ctx *fasthttp.RequestCtx) {

	ids := string(ctx.PostArgs().Peek("ids"))
	var id []string
	for _, v := range strings.Split(ids, ",") {
		if !validator.CtypeDigit(v) {
			helper.Print(ctx, false, helper.IDErr)
			return
		}
		id = append(id, v)
	}

	if len(id) == 0 {
		helper.Print(ctx, false, helper.IDErr)
		return
	}

	// 获取 CreditMemberLevel
	memberLevel, err := model.CreditMemberLevelByID(id[0])
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(memberLevel.CreditLevelID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	creditLevel, err := model.CreditLevelByID(memberLevel.CreditLevelID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(creditLevel.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	/*
		cateName, channelName, err := model.TunnelAndChannelGetName(creditLevel.CateID, creditLevel.ChannelID)
		if err != nil {
			helper.Print(ctx, false, err.Error())
			return
		}

		logMsg := fmt.Sprintf("批量删除【会员账号：%d;信用等级: %d；渠道名称:%s；通道名称:%s】",
			len(id), creditLevel.Level, cateName, channelName)
		defer model.SystemLogWrite(logMsg, ctx)
	*/

	err = model.MemberCreditLevelDelete(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}
