package controller

import (
	"fmt"

	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"

	"github.com/valyala/fasthttp"
)

type LockController struct{}

type memberLockParam struct {
	Username string `rule:"uname" min:"5" max:"14" msg:"username error" name:"username"` // 会员名
	Comment  string `rule:"none" min:"0" max:"50" msg:"comment error" name:"comment"`    // 备注
}

type memberLockListParam struct {
	Username  string `rule:"none" min:"1" msg:"username error" name:"username"` // 会员名
	LockName  string `rule:"none"  msg:"lock_name error" name:"lock_name"`      // 锁定操作人
	StartTime string `rule:"none" msg:"start_time error" name:"start_time"`
	EndTime   string `rule:"none" msg:"end_time error" name:"end_time"`
	Page      uint16 `rule:"digit" default:"1" min:"1" msg:"page error" name:"page"`
	PageSize  uint16 `rule:"digit" default:"10" min:"10" max:"200" msg:"page_size error" name:"page_size"`
}

// MemberInsert 财务管理-渠道管理-会员锁定-新增
func (that *LockController) MemberInsert(ctx *fasthttp.RequestCtx) {

	param := memberLockParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if param.Comment != "" {
		if !validator.CheckStringLength(param.Comment, 0, 50) {
			fmt.Println(err)
			helper.Print(ctx, false, helper.ParamErr)
			return
		}

		param.Comment = validator.FilterInjection(param.Comment)
	}

	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	member, err := model.MemberByUsername(param.Username)
	if err != nil {
		helper.Print(ctx, false, helper.UsernameErr)
		return
	}

	//

	// 写入系统日志
	//logMsg := fmt.Sprintf("锁定【会员账号: %s】", param.Username)
	//defer model.SystemLogWrite(logMsg, ctx)

	fields := map[string]string{
		"id":           helper.GenId(),
		"uid":          member.UID,
		"username":     param.Username,
		"state":        "1",
		"comment":      param.Comment,
		"created_uid":  admin["id"],
		"created_name": admin["name"],
		"created_at":   fmt.Sprintf("%d", ctx.Time().Unix()),
	}
	err = model.LockInsert(fields)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// MemberList 财务管理-渠道管理-会员锁定-列表
func (that *LockController) MemberList(ctx *fasthttp.RequestCtx) {

	param := memberLockListParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if param.Username != "" {
		if !validator.CheckUName(param.Username, 5, 14) {
			helper.Print(ctx, false, helper.UsernameErr)
			return
		}
	}

	if param.LockName != "" {
		if !validator.CheckAName(param.LockName, 5, 20) {
			helper.Print(ctx, false, helper.AdminNameErr)
			return
		}
	}

	data, err := model.LockList(param.Username, param.LockName, param.StartTime, param.EndTime, param.Page, param.PageSize)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// UpdateState 财务管理-渠道管理-会员锁定-启用
func (that *LockController) UpdateState(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	if !validator.CtypeDigit(id) {
		helper.Print(ctx, false, helper.IDErr)
		return
	}

	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	info, err := model.LockById(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if info.State == "0" {
		helper.Print(ctx, false, helper.NoDataUpdate)
		return
	}
	// 写入系统日志
	//logMsg := fmt.Sprintf("启用【会员账号: %s】", info.Username)
	//defer model.SystemLogWrite(logMsg, ctx)

	fields := map[string]string{
		"id":           id,
		"state":        "0",
		"updated_uid":  admin["id"],
		"updated_name": admin["name"],
		"updated_at":   fmt.Sprintf("%d", ctx.Time().Unix()),
	}
	err = model.LockUpdateState(info.UID, fields)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}
