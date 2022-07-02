package controller

import (
	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"
	"strconv"

	g "github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/shopspring/decimal"
	"github.com/valyala/fasthttp"
)

type DepositController struct{}

// 会员存款信息
type memberDepositParam struct {
	Username  string `rule:"alnum" min:"5" max:"14" msg:"username error" name:"username"`
	State     string `rule:"none" msg:"state error" name:"state"`
	ChannelID string `rule:"none" msg:"channel_id error" name:"channel_id"`
	TimeFlag  string `rule:"digit" min:"0" max:"2" default:"0" msg:"time_flag error" name:"time_flag"`
	StartTime string `rule:"none" msg:"start_time error" name:"start_time"`                   // 查询开始时间
	EndTime   string `rule:"none" msg:"end_time error" name:"end_time"`                       // 查询结束时间
	Page      int    `rule:"digit" default:"1" min:"1" msg:"page error" name:"page"`          // 页码
	PageSize  int    `rule:"digit" min:"10" max:"200" msg:"page_size error" name:"page_size"` // 页大小
}

// 补单审核
type depositReviewParam struct {
	ID     string `rule:"digit" msg:"id error" name:"id"`
	State  string `rule:"digit" name:"state" msg:"state error"`
	Remark string `rule:"none" name:"review_remark"`
	Code   string `rule:"digit" name:"code" msg:"code error"`
}

// 补单
type handDepositParam struct {
	ID         string `rule:"digit" msg:"id error" name:"id"`
	Remark     string `rule:"none" name:"remark"`
	RealAmount string `rule:"float" name:"real_amount"`
	Code       string `rule:"digit" name:"code" msg:"code error"`
}

// 下分补单
type reduceDepositParam struct {
	Username string `rule:"alnum" min:"5" max:"14" msg:"username error" name:"username"`
	Remark   string `rule:"none" name:"remark"`
	Amount   string `rule:"float" name:"amount"`
	Code     string `rule:"digit" name:"code" msg:"code error"`
}

// 订单列表
type depositListParam struct {
	ID         string `rule:"none" name:"id"`
	Username   string `rule:"none" msg:"username error" name:"username"`
	ParentName string `rule:"none" msg:"parent_name error" name:"parent_name"`
	GroupName  string `rule:"none" msg:"group_name error" name:"group_name"`
	State      int    `rule:"none" default:"0" name:"state"`
	OID        string `rule:"none" name:"oid"`
	CID        string `rule:"none" json:"cid"`
	ChannelID  string `rule:"none" name:"channel_id"`
	Automatic  string `rule:"none" name:"automatic"`
	Flag       int    `rule:"digit" name:"flag" min:"1" max:"2"`                               //1=下分历史和下分列表
	TimeFlag   uint8  `rule:"digit" default:"1" min:"0" max:"1" name:"time_flag"`              // 时间类型  1:创建时间 0:完成时间
	StartTime  string `rule:"none" msg:"start_time error" name:"start_time"`                   // 查询开始时间
	EndTime    string `rule:"none" msg:"end_time error" name:"end_time"`                       // 查询结束时间
	MinAmount  string `rule:"none" msg:"min_amount error" name:"min_amount"`                   //
	MaxAmount  string `rule:"none" msg:"max_amount error" name:"max_amount"`                   //
	Page       int    `rule:"digit" default:"1" min:"1" msg:"page error" name:"page"`          // 页码
	PageSize   int    `rule:"digit" min:"10" max:"200" msg:"page_size error" name:"page_size"` // 页大小
	Ty         int    `rule:"digit" min:"0" max:"4" default:"0" name:"ty"`                     // 1 三方订单 2 usdt 订单 3 线下转卡 4 线下转USDT
	Dty        int    `rule:"none" default:"0" name:"dty"`
}

//Detail 会员列表-存款信息
func (that *DepositController) Detail(ctx *fasthttp.RequestCtx) {

	param := memberDepositParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	data, err := model.DepositDetail(param.Username, param.State, param.ChannelID,
		param.TimeFlag, param.StartTime, param.EndTime, param.Page, param.PageSize)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

//Review 补单审核-审核
func (that *DepositController) Review(ctx *fasthttp.RequestCtx) {

	param := depositReviewParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	/*
		deposit, err := model.DepositFindOne(param.ID)
		if err != nil {
			helper.Print(ctx, false, err.Error())
			return
		}


			keyword := "通过"
			if param.State == strconv.Itoa(model.DepositCancelled) {
				keyword = "拒绝"
			}

			// 写入系统日志
			logMsg := fmt.Sprintf("%s【订单号: %s；会员账号: %s；金额: %.4f；审核时间: %s】",
				keyword, param.ID, deposit.Username, deposit.Amount, model.TimeFormat(ctx.Time().Unix()))
			defer model.SystemLogWrite(logMsg, ctx)
	*/

	err = model.DepositReview(param.ID, param.Remark, param.State, admin["name"], admin["id"])
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// History 存款历史记录
func (that *DepositController) History(ctx *fasthttp.RequestCtx) {

	param := depositListParam{}
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

	if param.ID != "" {
		if !validator.CheckStringDigit(param.ID) {
			helper.Print(ctx, false, helper.IDErr)
			return
		}
	}

	if param.OID != "" {
		if !validator.CheckStringAlnum(param.OID) {
			helper.Print(ctx, false, helper.OIDErr)
			return
		}
	}

	if param.CID != "" {
		if !validator.CheckStringAlnum(param.CID) {
			helper.Print(ctx, false, helper.ParamErr)
			return
		}
	}

	if param.ChannelID != "" {
		if !validator.CheckStringDigit(param.ChannelID) {
			helper.Print(ctx, false, helper.ChannelIDErr)
			return
		}
	}

	if param.State != 0 {
		if param.State != model.DepositSuccess && param.State != model.DepositCancelled {
			helper.Print(ctx, false, helper.StateParamErr)
			return
		}
	}

	if param.StartTime == "" || param.EndTime == "" {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	data, err := model.DepositHistory(param.Username, param.ParentName, param.GroupName, param.ID, param.ChannelID, param.OID, strconv.Itoa(param.State),
		param.MinAmount, param.MaxAmount, param.StartTime, param.EndTime, param.CID, param.TimeFlag, param.Flag, param.Page, param.PageSize, param.Ty, param.Dty)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// List 存款列表  三方存款
func (that *DepositController) List(ctx *fasthttp.RequestCtx) {

	param := depositListParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	ex := g.Ex{
		"amount": g.Op{"gte": 0.00},
		"flag":   model.DepositFlagThird,
	}

	if param.Username != "" {
		if !validator.CheckUName(param.Username, 5, 14) {
			helper.Print(ctx, false, helper.UsernameErr)
			return
		}

		ex["username"] = param.Username
	}

	if param.ID != "" {
		if !validator.CheckStringDigit(param.ID) {
			helper.Print(ctx, false, helper.IDErr)
			return
		}

		ex["id"] = param.ID
	}

	if param.OID != "" {
		if !validator.CheckStringAlnum(param.OID) {
			helper.Print(ctx, false, helper.OIDErr)
			return
		}

		ex["oid"] = param.OID
	}

	if param.ChannelID != "" {
		if !validator.CheckStringDigit(param.ChannelID) {
			helper.Print(ctx, false, helper.ChannelIDErr)
			return
		}

		ex["channel_id"] = param.ChannelID
	}

	if param.State != 0 {
		if param.State < model.DepositConfirming || param.State > model.DepositReviewing {
			helper.Print(ctx, false, helper.StateParamErr)
			return
		}

		ex["state"] = param.State
	}

	if param.Flag == 1 {
		ex["amount"] = g.Op{"lt": 0.00}
	}

	if param.Automatic != "" {
		ex["automatic"] = param.Automatic
	}

	if param.MinAmount != "" && param.MaxAmount != "" {
		ex["amount"] = g.Op{"between": exp.NewRangeVal(param.MinAmount, param.MaxAmount)}
	}

	data, err := model.DepositList(ex, param.StartTime, param.EndTime, param.Page, param.PageSize)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// USDTList 存款列表 USDT
func (that *DepositController) USDTList(ctx *fasthttp.RequestCtx) {

	username := string(ctx.PostArgs().Peek("username"))
	id := string(ctx.PostArgs().Peek("id"))
	oid := string(ctx.PostArgs().Peek("oid"))
	protocolType := string(ctx.PostArgs().Peek("protocol_type"))
	minAmount := string(ctx.PostArgs().Peek("min_amount"))
	maxAmount := string(ctx.PostArgs().Peek("max_amount"))
	startTime := string(ctx.PostArgs().Peek("start_time"))
	endTime := string(ctx.PostArgs().Peek("end_time"))
	page := ctx.PostArgs().GetUintOrZero("page")
	pageSize := ctx.PostArgs().GetUintOrZero("page_size")

	if page == 0 {
		page = 1
	}

	if pageSize == 0 {
		pageSize = 10
	}

	ex := g.Ex{
		"flag":  model.DepositFlagThirdUSTD,
		"state": model.DepositConfirming,
	}

	if username != "" {
		if !validator.CheckUName(username, 5, 14) {
			helper.Print(ctx, false, helper.UsernameErr)
			return
		}

		ex["username"] = username
	}

	if protocolType != "" {
		ex["protocol_type"] = protocolType
	}

	if id != "" {
		if !validator.CheckStringDigit(id) {
			helper.Print(ctx, false, helper.IDErr)
			return
		}

		ex["id"] = id
	}

	if oid != "" {
		if !validator.CheckStringAlnum(oid) {
			helper.Print(ctx, false, helper.OIDErr)
			return
		}

		ex["oid"] = oid
	}

	if minAmount != "" && maxAmount != "" {
		ex["usdt_apply_amount"] = g.Op{"between": exp.NewRangeVal(minAmount, maxAmount)}
	}

	data, err := model.DepositList(ex, startTime, endTime, page, pageSize)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// Manual 补单
func (that *DepositController) Manual(ctx *fasthttp.RequestCtx) {

	param := handDepositParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if param.Remark == "" {
		helper.Print(ctx, false, helper.RemarkFMTErr)
		return
	}

	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	deposit, err := model.DepositFindOne(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	channel, err := model.TunnelByID(deposit.ChannelID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if len(channel.ID) == 0 {
		helper.Print(ctx, false, helper.ChannelNotExist)
		return
	}

	/*
		// 写入系统日志
		logMsg := fmt.Sprintf("补单【订单号: %s；会员账号: %s；渠道单号: %s；通道名称: %s；订单金额: %.4f；到账金额: %s; 订单时间: %s；补单时间: %s】",
			param.ID, deposit.Username, deposit.OID, channel.Name, deposit.Amount, param.RealAmount, model.TimeFormat(deposit.CreatedAt),
			model.TimeFormat(ctx.Time().Unix()))
		defer model.SystemLogWrite(logMsg, ctx)
	*/

	err = model.DepositManual(param.ID, param.RealAmount, param.Remark, admin["name"], admin["id"])
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// Reduce 财务管理-下分
func (that *DepositController) Reduce(ctx *fasthttp.RequestCtx) {

	param := reduceDepositParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if param.Remark == "" {
		helper.Print(ctx, false, helper.RemarkFMTErr)
		return
	}

	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	/*
		// 写入系统日志
		logMsg := fmt.Sprintf("提交【会员账号:%s；调整金额:%s】", param.Username, param.Amount)
		defer model.SystemLogWrite(logMsg, ctx)
	*/

	err = model.DepositReduce(param.Username, param.Amount, param.Remark, admin["name"], admin["id"])
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// OfflineUSDT USDT修改订单状态 确认金额
func (that *DepositController) OfflineUSDT(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	remark := string(ctx.PostArgs().Peek("remark"))
	usdtAmount := ctx.PostArgs().GetUfloatOrZero("amount")

	if !validator.CheckStringDigit(id) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if remark != "" {
		remark = validator.FilterInjection(remark)
	}

	if usdtAmount <= 0 {
		helper.Print(ctx, false, helper.AmountErr)
		return
	}

	deposit, err := model.DepositFindOne(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if deposit.State != model.DepositConfirming {
		helper.Print(ctx, false, helper.RecordNotExistErr)
		return
	}

	/*
		// 写入系统日志
		logMsg := fmt.Sprintf("线下转卡【订单id:%s；到账金额USDT:%.4f】", id, usdtAmount)
		defer model.SystemLogWrite(logMsg, ctx)
	*/

	usdt_info_temp, err := model.UsdtInfo()
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	usdt_rate, err := decimal.NewFromString(usdt_info_temp["usdt_rate"])
	if err != nil {
		helper.Print(ctx, false, helper.AmountErr)
		return
	}

	// 计算获取上分的越南盾金额 单位kvnd
	amount := decimal.NewFromFloat(usdtAmount).Mul(usdt_rate).
		DivRound(decimal.NewFromInt(1000), 3).String()

	rec := g.Record{
		"usdt_final_amount": usdtAmount,
		"amount":            amount,
		"review_remark":     remark,
		"state":             model.DepositReviewing,
	}

	err = model.DepositRecordUpdate(id, rec)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// OfflineUSDTReview 线下转卡-审核
func (that *DepositController) OfflineUSDTReview(ctx *fasthttp.RequestCtx) {

	remark := string(ctx.PostArgs().Peek("remark"))
	state := ctx.PostArgs().GetUintOrZero("state")
	id := string(ctx.PostArgs().Peek("id"))

	if remark == "" || !validator.CheckStringDigit(id) {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	if state != model.DepositSuccess && state != model.DepositCancelled {
		helper.Print(ctx, false, helper.StateParamErr)
		return
	}

	admin, err := model.AdminToken(ctx)
	if err != nil || len(admin["id"]) < 1 {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	deposit, err := model.DepositFindOne(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	/*
		keyword := "通过"
		if state == model.DepositCancelled {
			keyword = "拒绝"
		}

		// 写入系统日志
		logMsg := fmt.Sprintf("线下转卡:%s【订单号: %s；会员账号: %s；金额: %.4f；审核时间: %s】",
			keyword, id, deposit.Username, deposit.Amount, model.TimeFormat(ctx.Time().Unix()))
		defer model.SystemLogWrite(logMsg, ctx)
	*/

	err = model.DepositUSDTReview(id, remark, admin["name"], admin["id"], deposit.UID, state)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}
