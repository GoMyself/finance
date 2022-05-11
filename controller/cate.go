package controller

import (
	"fmt"

	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"

	"github.com/valyala/fasthttp"
)

type CateController struct{}

type cateParam struct {
	ID       string `rule:"digit" default:"0" msg:"id error" name:"id"`
	CateName string `rule:"chnAlnum" min:"1" max:"20" msg:"cate_name error" name:"cate_name"` // 渠道名称
	Comment  string `rule:"none" msg:"comment error" name:"comment"`                          // 备注
	Code     string `rule:"digit" msg:"code error" name:"code"`                               // 动态验证码
}

type cateListParam struct {
	All      string `rule:"digit" min:"0" max:"1" default:"0" msg:"all error" name:"all"` // 商户id
	CateName string `rule:"none" msg:"cate_name error" name:"cate_name"`                  // 渠道名称
}

type cateStateParam struct {
	ID    string `rule:"digit" default:"0" msg:"id error" name:"id"`
	State string `rule:"digit" min:"0" max:"1" msg:"state error" name:"state"` // 0:关闭1:开启
	Code  string `rule:"digit" msg:"code error" name:"code"`                   // 动态验证码
}

// List 财务管理-渠道管理-列表
func (that *CateController) List(ctx *fasthttp.RequestCtx) {

	param := cateListParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if param.CateName != "" {
		if !validator.CheckStringCHNAlnum(param.CateName) || !validator.CheckStringLength(param.CateName, 1, 20) {
			helper.Print(ctx, false, helper.CateNameErr)
			return
		}
	}

	data, err := model.CateList(param.CateName, param.All)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

func (that *CateController) Cache(ctx *fasthttp.RequestCtx) {
	data := model.CateListRedis()
	helper.PrintJson(ctx, true, data)
}

// Withdraw 财务管理-提款通道
func (that *CateController) Withdraw(ctx *fasthttp.RequestCtx) {

	amount := ctx.PostArgs().GetUfloatOrZero("amount")

	data, err := model.CateWithdrawList(amount)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// Insert 财务管理-渠道管理-新增
func (that *CateController) Insert(ctx *fasthttp.RequestCtx) {

	param := cateParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	//content := fmt.Sprintf("新增【渠道名称: %s】", param.CateName)
	//defer model.SystemLogWrite(content, ctx)

	if param.Comment != "" {
		if !validator.CheckStringLength(param.Comment, 0, 20) {
			helper.Print(ctx, false, helper.RemarkFMTErr)
			return
		}
	}

	fields := map[string]string{
		"name":       param.CateName,
		"comment":    param.Comment,
		"created_at": fmt.Sprintf("%d", ctx.Time().Unix()),
	}

	err = model.CateInsert(fields)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// Update 财务管理-渠道管理-修改
func (that *CateController) Update(ctx *fasthttp.RequestCtx) {

	param := cateParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	//content := fmt.Sprintf("编辑【渠道名称: %s】", param.CateName)
	//defer model.SystemLogWrite(content, ctx)

	if param.ID == "0" {
		helper.Print(ctx, false, helper.IDErr)
		return
	}

	if param.Comment != "" {
		if !validator.CheckStringLength(param.Comment, 0, 20) {
			helper.Print(ctx, false, helper.RemarkFMTErr)
			return
		}
	}

	fields := map[string]string{
		"id":      param.ID,
		"name":    param.CateName,
		"comment": param.Comment,
	}
	err = model.CateUpdate(fields)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// Delete 财务管理-渠道管理-删除
func (that *CateController) Delete(ctx *fasthttp.RequestCtx) {

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

	err := model.CateDelete(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// UpdateState 财务管理-渠道管理-启用/停用
func (that *CateController) UpdateState(ctx *fasthttp.RequestCtx) {

	param := cateStateParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	cate, err := model.CateByID(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(cate.ID) == 0 {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	if cate.State == param.State {
		helper.Print(ctx, false, helper.NoDataUpdate)
		return
	}

	/*
		keyword := "开启"
		if param.State == "0" {
			keyword = "关闭"
		}
		content := fmt.Sprintf("%s【商户ID: %s ；渠道名称: %s】", keyword, cate.MerchantId, cate.Name)
		defer model.SystemLogWrite(content, ctx)
	*/
	err = model.CateSet(param.ID, param.State)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}
