package controller

import (
	"finance/contrib/helper"
	"finance/model"
	"fmt"
	"github.com/shopspring/decimal"
	"github.com/valyala/fasthttp"
	"strings"
)

type UsdtController struct{}

// SetRate 设置usdt汇率
func (that *UsdtController) SetRate(ctx *fasthttp.RequestCtx) {

	r := string(ctx.PostArgs().Peek("rate"))
	rate, err := decimal.NewFromString(r)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	c := &model.Config{
		Name:    "usdt_rate",
		Content: rate.Truncate(4).String(),
	}

	err = model.ConfigSet(c)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	content := fmt.Sprintf("汇率编辑[汇率:%s]", rate.String())
	defer model.SystemLogWrite(content, ctx)

	helper.Print(ctx, true, helper.Success)
}

func (that *UsdtController) SetTRC(ctx *fasthttp.RequestCtx) {

	addr := string(ctx.PostArgs().Peek("addr"))
	if addr == "" || !strings.HasPrefix(addr, "T") {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	c := &model.Config{
		Name:    "trc_addr",
		Content: addr,
	}
	err := model.ConfigSet(c)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	content := fmt.Sprintf("设置TRC地址[%s]", addr)
	defer model.SystemLogWrite(content, ctx)

	helper.Print(ctx, true, helper.Success)
}

// GetRate 设置usdt汇率
func (that *UsdtController) GetRate(ctx *fasthttp.RequestCtx) {

	rate, err := model.USDTConfig()
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, rate.String())
}

// GetTRC 获取TRC地址
func (that *UsdtController) GetTRC(ctx *fasthttp.RequestCtx) {

	addr, err := model.TRCConfig()
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, addr)
}
