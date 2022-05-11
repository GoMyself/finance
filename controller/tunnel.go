package controller

import (
	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"

	"github.com/valyala/fasthttp"
)

type TunnelController struct{}

type updateTunnelParam struct {
	ID   string `rule:"digit" msg:"id error" name:"id"`                      // id
	Sort string `rule:"digit" min:"1" max:"99" msg:"sort error" name:"sort"` // 排序
	Code string `rule:"digit" msg:"code error" name:"code"`                  // 动态验证码
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

	param := updateTunnelParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamNull)
		return
	}

	tunnel, err := model.TunnelByID(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(tunnel.ID) == 0 {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	//content := fmt.Sprintf("编辑【通道名称: %s】", tunnel.Name)
	//defer model.SystemLogWrite(content, ctx)

	fields := map[string]string{
		"id":   param.ID,
		"sort": param.Sort,
	}

	err = model.TunnelUpdate(fields)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}
