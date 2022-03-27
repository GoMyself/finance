package controller

import (
	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
)

type PromoController struct{}

type promoStateParam struct {
	ID    string `rule:"digit" default:"0" msg:"id error" name:"id"`
	State string `rule:"digit" min:"0" max:"1" msg:"state error" name:"state"` // 0:关闭1:开启
}

func (that *PromoController) Detail(ctx *fasthttp.RequestCtx) {

	id := string(ctx.QueryArgs().Peek("id"))
	if !helper.CtypeDigit(id) {
		helper.Print(ctx, false, helper.IDErr)
		return
	}

	res, err := model.PromoDetail(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.PrintJson(ctx, true, res)
}

func (that *PromoController) UpdateState(ctx *fasthttp.RequestCtx) {

	param := promoStateParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	recs := g.Record{
		"promo_state": param.State,
	}
	err = model.PromoUpdate(recs, param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

func (that *PromoController) UpdateQuota(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	content := string(ctx.PostArgs().Peek("content"))
	if !helper.CtypeDigit(id) {
		helper.Print(ctx, false, helper.IDErr)
		return
	}

	err := fastjson.Validate(content)
	if err != nil {
		helper.Print(ctx, false, helper.RemarkFMTErr)
		return
	}

	tunnel, err := model.TunnelByID(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(tunnel.ID) == 0 {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	logMsg := fmt.Sprintf("设置【通道名称: %s】", tunnel.Name)
	defer model.SystemLogWrite(logMsg, ctx)

	recs := g.Record{
		"content": content,
	}

	err = model.PromoUpdate(recs, id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}
