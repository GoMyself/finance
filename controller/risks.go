package controller

import (
	"finance/contrib/helper"
	"finance/model"
	"github.com/valyala/fasthttp"
)

type RisksController struct{}

// 关闭风控自动派单
func (that *RisksController) CloseAuto(ctx *fasthttp.RequestCtx) {

	uid := string(ctx.QueryArgs().Peek("uid"))
	if uid == "0" {
		data, err := model.AdminToken(ctx)
		if err != nil {
			helper.Print(ctx, false, helper.AccessTokenExpires)
			return
		}

		uid = data["id"]
	}

	err := model.RisksCloseAuto(uid)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// 开启风控自动派单
func (that *RisksController) OpenAuto(ctx *fasthttp.RequestCtx) {

	uid := string(ctx.QueryArgs().Peek("uid"))
	if uid == "0" {
		data, err := model.AdminToken(ctx)
		if err != nil {
			helper.Print(ctx, false, helper.AccessTokenExpires)
			return
		}

		uid = data["id"]
	}

	err := model.RisksOpenAuto(uid)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

//查询开启自动接单的人员列表
func (that *RisksController) List(ctx *fasthttp.RequestCtx) {

	list, err := model.RisksList()
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, list)
}

func (that *RisksController) State(ctx *fasthttp.RequestCtx) {

	uid := string(ctx.QueryArgs().Peek("uid"))
	if uid != "" {
		data, err := model.AdminToken(ctx)
		if err != nil {
			helper.Print(ctx, false, helper.AccessTokenExpires)
			return
		}

		uid = data["id"]
	}

	state := model.IsExistRisks(uid)

	helper.Print(ctx, true, state)
}

func (that *RisksController) SetNumber(ctx *fasthttp.RequestCtx) {

	num := string(ctx.QueryArgs().Peek("num"))
	err := model.SetOrderNum(num)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

//查询风控人员列表
func (that *RisksController) Receives(ctx *fasthttp.RequestCtx) {

	list, err := model.RisksReceives()
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, list)
}

//查询开启自动接单的人员列表
func (that *RisksController) Number(ctx *fasthttp.RequestCtx) {

	num, _ := model.RisksNumber()

	helper.Print(ctx, true, num)
}

func (that *RisksController) SetRegMax(ctx *fasthttp.RequestCtx) {

	num := string(ctx.PostArgs().Peek("num"))
	err := model.SetRegMax(num)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

func (that *RisksController) RegMax(ctx *fasthttp.RequestCtx) {

	num, _ := model.RisksRegMax()

	helper.Print(ctx, true, num)
}
