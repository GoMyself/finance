package controller

import (
	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"

	"github.com/valyala/fasthttp"
)

type TunnelController struct{}

type tunnelUpdateParam struct {
	ID    string `rule:"digit" msg:"id error" name:"id"`                        // id
	Sort  string `rule:"digit" min:"1" max:"99" msg:"sort error" name:"sort"`   // 排序
	Value string `rule:"float" min:"1" max:"99" msg:"value error" name:"value"` // 排序
	State string `rule:"digit" min:"0" max:"1" msg:"state error" name:"state"`  // 排序
	Code  string `rule:"digit" msg:"code error" name:"code"`                    // 动态验证码
}

// List 财务管理-渠道管理-通道类型管理-列表
func (that *TunnelController) List(ctx *fasthttp.RequestCtx) {

	data, err := model.TunnelList()
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// Update 财务管理-渠道管理-通道类型管理-修改
func (that *TunnelController) Update(ctx *fasthttp.RequestCtx) {

	param := tunnelUpdateParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamNull)
		return
	}

	if !validator.CheckIntScope(param.Value, -99, 99) {
		helper.Print(ctx, false, helper.AmountOutRange)
		return
	}
	//content := fmt.Sprintf("编辑【通道名称: %s】", tunnel.Name)
	//defer model.SystemLogWrite(content, ctx)
	if param.ID == "7" && param.Value != "0" {
		helper.Print(ctx, false, helper.Blocked)
		return
	}

	err = model.TunnelUpdate(param.ID, param.State, param.Value, param.Sort)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}
