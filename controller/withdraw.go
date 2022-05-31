package controller

import (
	"database/sql"
	"finance/contrib/helper"
	"finance/contrib/validator"
	"finance/model"
	"fmt"
	"strconv"
	"strings"

	g "github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/shopspring/decimal"
	"github.com/valyala/fasthttp"
)

type WithdrawController struct{}

// 会员申请提现
type memberWithdrawParam struct {
	BID    string `name:"bid" rule:"digit" msg:"bid error"` // 下发的银行卡或虚拟钱包的ID
	Amount string `name:"amount" rule:"digit" min:"200" max:"1000000" msg:"amount error[200-1000000]"`
}

// 提现拒绝
type withdrawReviewReject struct {
	ID             string `name:"id" rule:"digit" msg:"id error"`
	Remark         string `name:"remark" rule:"filter" default:"" min:"0" max:"50" msg:"remark error"`
	WithdrawRemark string `name:"withdraw_remark" rule:"filter" default:"" min:"0" max:"50" msg:"withdraw_remark error"`
}

// 提款审核
type withdrawReviewParam struct {
	ID       string `name:"id" rule:"digit" msg:"id error"`
	Ty       uint8  `name:"ty" rule:"digit" min:"1" max:"2" msg:"ty error"` // 1手动代付 2 手动出款
	Pid      string `name:"pid" rule:"digit" default:"0" msg:"pid error"`
	Remark   string `name:"remark" rule:"filter" default:"" min:"0" max:"50" msg:"remark error"`
	BankId   string `name:"bank_id" rule:"none" msg:"bank_id error"`
	BankName string `name:"bank_name" rule:"none"`
	RealName string `name:"real_name" rule:"none"`
	CardNo   string `name:"card_no" rule:"none"`
}

//  风控审核 拒绝
type withdrawRiskRejectParam struct {
	ID             string `name:"id" rule:"digit" msg:"id error"`
	ReviewRemark   string `name:"review_remark" rule:"filter" min:"1" max:"100" msg:"review_remark error"`
	WithdrawRemark string `name:"withdraw_remark" rule:"filter" default:"" min:"1" max:"100" msg:"withdraw remark error"`
}

// 订单挂起
type withdrawHangUp struct {
	ID           string `name:"id" rule:"digit" msg:"id error"`
	RemarkID     string `name:"remark_id" rule:"digit" msg:"remark_id error"`
	HangUpRemark string `name:"hang_up_remark" rule:"filter" min:"1" max:"100" msg:"hang_up_remark error"`
}

type withdrawRecord struct {
	Success int    `json:"success"`
	Fail    int    `json:"fail"`
	BindAt  uint64 `json:"bind_at"`
}

// Withdraw 会员申请提现
func (that *WithdrawController) Withdraw(ctx *fasthttp.RequestCtx) {

	param := memberWithdrawParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	id, err := model.WithdrawUserInsert(param.Amount, param.BID, ctx)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, id)
}

// RiskReviewList 风控待审核列表
func (that *WithdrawController) RiskReviewList(ctx *fasthttp.RequestCtx) {

	id := string(ctx.FormValue("id"))                    // 订单号
	realName := string(ctx.FormValue("real_name"))       // 真实姓名
	confirmName := string(ctx.FormValue("confirm_name")) // 领取人
	page, err := strconv.ParseUint(string(ctx.FormValue("page")), 10, 64)
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.ParseUint(string(ctx.FormValue("page_size")), 10, 64)
	if err != nil {
		pageSize = 15
	}

	ex := g.Ex{}
	if realName != "" {
		if len([]rune(id)) > 30 {
			helper.Print(ctx, false, helper.RealNameFMTErr)
			return
		}

		ex["real_name_hash"] = realName
	}

	if confirmName != "" {
		if !validator.CheckStringAlnum(confirmName) {
			helper.Print(ctx, false, helper.AdminNameErr)
			return
		}
		ex["confirm_name"] = confirmName
	}

	if id != "" {
		if !validator.CheckStringDigit(id) {
			helper.Print(ctx, false, helper.IDErr)
			return
		}
		ex = g.Ex{"id": id}
	}

	// 已派单
	ex["state"] = model.WithdrawDispatched
	data, err := model.WithdrawList(ex, 3, "", "", uint(page), uint(pageSize))
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	result, err := model.WithdrawDealListData(data)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, result)
}

// RiskWaitConfirmList 风控待领取列表
func (that *WithdrawController) RiskWaitConfirmList(ctx *fasthttp.RequestCtx) {

	id := string(ctx.FormValue("id"))              // 订单号
	realName := string(ctx.FormValue("real_name")) // 真实姓名
	startTime := string(ctx.FormValue("start_time"))
	endTime := string(ctx.FormValue("end_time"))
	username := string(ctx.FormValue("username"))
	maxAmount := string(ctx.FormValue("max_amount"))
	minAmount := string(ctx.FormValue("min_amount"))
	vips := string(ctx.FormValue("vips"))
	page, err := strconv.ParseUint(string(ctx.FormValue("page")), 10, 64)
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.ParseUint(string(ctx.FormValue("page_size")), 10, 64)
	if err != nil {
		pageSize = 15
	}

	if startTime == "" || endTime == "" {
		helper.Print(ctx, false, helper.DateTimeErr)
		return
	}

	ex := g.Ex{}
	if realName != "" {
		if len([]rune(id)) > 30 {
			helper.Print(ctx, false, helper.RealNameFMTErr)
			return
		}

		ex["real_name_hash"] = realName
	}

	if username != "" {
		if !validator.CheckUName(username, 4, 9) {
			helper.Print(ctx, false, helper.UsernameErr)
			return
		}

		ex["username"] = username
	}

	if minAmount != "" && maxAmount != "" {

		if !validator.CheckStringDigit(minAmount) || !validator.CheckStringDigit(maxAmount) {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}

		minAmountInt, _ := strconv.Atoi(minAmount)
		maxAmountInt, _ := strconv.Atoi(maxAmount)
		if minAmountInt > maxAmountInt {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}

		ex["amount"] = g.Op{"between": exp.NewRangeVal(minAmountInt, maxAmountInt)}
	}

	if vips != "" {
		vipSlice := strings.Split(vips, ",")
		for _, v := range vipSlice {
			if !validator.CheckStringDigit(v) || !validator.CheckIntScope(v, 1, 11) {
				helper.Print(ctx, false, helper.MemberLevelErr)
				return
			}
		}
		ex["level"] = vipSlice
	}

	if id != "" {
		if !validator.CheckStringDigit(id) {
			helper.Print(ctx, false, helper.IDErr)
			return
		}
		ex = g.Ex{"id": id}
	}
	// 待派单
	ex["state"] = model.WithdrawReviewing

	data, err := model.WithdrawList(ex, 1, startTime, endTime, uint(page), uint(pageSize))
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	result, err := model.WithdrawDealListData(data)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, result)
}

// HangUpList 风控审核挂起列表
func (that *WithdrawController) HangUpList(ctx *fasthttp.RequestCtx) {

	id := string(ctx.FormValue("id"))              // 订单号
	realName := string(ctx.FormValue("real_name")) // 真实姓名
	startTime := string(ctx.FormValue("start_time"))
	endTime := string(ctx.FormValue("end_time"))
	maxAmount := string(ctx.FormValue("max_amount"))
	minAmount := string(ctx.FormValue("min_amount"))
	username := string(ctx.FormValue("username"))
	remarkID := string(ctx.FormValue("remark_id"))
	hangUpName := string(ctx.FormValue("hang_up_name"))
	vips := string(ctx.FormValue("vips"))
	page, err := strconv.ParseUint(string(ctx.FormValue("page")), 10, 64)
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.ParseUint(string(ctx.FormValue("page_size")), 10, 64)
	if err != nil {
		pageSize = 15
	}

	if startTime == "" || endTime == "" {
		helper.Print(ctx, false, helper.DateTimeErr)
		return
	}

	ex := g.Ex{}
	if realName != "" {
		if len([]rune(id)) > 30 {
			helper.Print(ctx, false, helper.RealNameFMTErr)
			return
		}

		ex["real_name_hash"] = realName
	}

	if minAmount != "" && maxAmount != "" {
		if !validator.CheckStringDigit(minAmount) || !validator.CheckStringDigit(maxAmount) {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}

		minAmountInt, _ := strconv.Atoi(minAmount)
		maxAmountInt, _ := strconv.Atoi(maxAmount)
		if minAmountInt > maxAmountInt {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}

		ex["amount"] = g.Op{"between": exp.NewRangeVal(minAmountInt, maxAmountInt)}
	}

	if username != "" {
		if !validator.CheckUName(username, 4, 9) {
			helper.Print(ctx, false, helper.UsernameErr)
			return
		}

		ex["username"] = username
	}

	if remarkID != "" {
		ex["remark_id"] = remarkID
	}

	if hangUpName != "" {
		if !validator.CheckAName(hangUpName, 5, 20) {
			helper.Print(ctx, false, helper.AdminNameErr)
			return
		}

		ex["hang_up_name"] = hangUpName
	}

	if vips != "" {
		vipSlice := strings.Split(vips, ",")
		for _, v := range vipSlice {
			if !validator.CheckStringDigit(v) || !validator.CheckIntScope(v, 1, 11) {
				helper.Print(ctx, false, helper.MemberLevelErr)
				return
			}
		}
		ex["level"] = vipSlice
	}

	if id != "" {
		if !validator.CheckStringDigit(id) {
			helper.Print(ctx, false, helper.IDErr)
			return
		}

		ex = g.Ex{"id": id}
	}

	// 挂起
	ex["state"] = model.WithdrawHangup
	data, err := model.WithdrawList(ex, 1, startTime, endTime, uint(page), uint(pageSize))
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	result, err := model.WithdrawDealListData(data)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, result)
}

// RiskHistory 历史记录列表
func (that *WithdrawController) RiskHistory(ctx *fasthttp.RequestCtx) {

	id := string(ctx.FormValue("id"))              // 订单号
	realName := string(ctx.FormValue("real_name")) // 真实姓名
	startTime := string(ctx.FormValue("start_time"))
	endTime := string(ctx.FormValue("end_time"))
	maxAmount := string(ctx.FormValue("max_amount"))
	minAmount := string(ctx.FormValue("min_amount"))
	username := string(ctx.FormValue("username"))
	confirmName := string(ctx.FormValue("confirm_name"))
	vips := string(ctx.FormValue("vips"))
	state := string(ctx.FormValue("state"))
	page, err := strconv.ParseUint(string(ctx.FormValue("page")), 10, 64)
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.ParseUint(string(ctx.FormValue("page_size")), 10, 64)
	if err != nil {
		pageSize = 15
	}

	if startTime == "" || endTime == "" {
		helper.Print(ctx, false, helper.DateTimeErr)
		return
	}

	ex := make(map[string]interface{})
	rangeParam := map[string][]interface{}{}
	if realName != "" {
		if len([]rune(id)) > 30 {
			helper.Print(ctx, false, helper.RealNameFMTErr)
			return
		}

		ex["real_name_hash"] = realName
	}

	if minAmount != "" && maxAmount != "" {
		if !validator.CheckStringDigit(minAmount) || !validator.CheckStringDigit(maxAmount) {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}

		minAmountInt, _ := strconv.Atoi(minAmount)
		maxAmountInt, _ := strconv.Atoi(maxAmount)
		if minAmountInt > maxAmountInt {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}

		rangeParam["amount"] = []interface{}{minAmountInt, maxAmountInt}
	}

	if username != "" {
		if !validator.CheckUName(username, 4, 9) {
			helper.Print(ctx, false, helper.UsernameErr)
			return
		}

		ex["username"] = username
	}

	if confirmName != "" {
		if !validator.CheckStringAlnum(confirmName) {
			helper.Print(ctx, false, helper.AdminNameErr)
			return
		}

		ex["confirm_name"] = confirmName
	}

	if vips != "" {
		vipSlice := strings.Split(vips, ",")
		for _, v := range vipSlice {
			if !validator.CheckStringDigit(v) || !validator.CheckIntScope(v, 1, 11) {
				helper.Print(ctx, false, helper.MemberLevelErr)
				return
			}
		}

		var vipInterface []interface{}
		for _, v := range vipSlice {
			vipInterface = append(vipInterface, v)
		}
		ex["level"] = vipInterface
	}

	baseState := []interface{}{
		model.WithdrawReviewReject,
		model.WithdrawDealing,
		model.WithdrawSuccess,
		model.WithdrawFailed,
		model.WithdrawAbnormal,
		model.WithdrawAutoPayFailed,
	}
	ex["state"] = baseState

	if state != "" {
		stateInt, err := strconv.Atoi(state)
		if err != nil {
			helper.Print(ctx, false, helper.StateParamErr)
			return
		}

		if stateInt < model.WithdrawReviewReject && stateInt > model.WithdrawAutoPayFailed {
			helper.Print(ctx, false, helper.StateParamErr)
			return
		}

		ex["state"] = state
	}

	if id != "" {
		if !validator.CheckStringDigit(id) {
			helper.Print(ctx, false, helper.IDErr)
			return
		}

		ex = map[string]interface{}{
			"_id":   id,
			"state": baseState,
		}
	}

	data, err := model.WithdrawHistoryList(ex, rangeParam, 1, startTime, endTime, uint(page), uint(pageSize))
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	result, err := model.WithdrawDealListData(data)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, result)
}

// HistoryList 后台提款列表
func (that *WithdrawController) HistoryList(ctx *fasthttp.RequestCtx) {

	id := string(ctx.FormValue("id")) // 订单号
	username := string(ctx.FormValue("username"))
	startTime := string(ctx.FormValue("start_time"))
	endTime := string(ctx.FormValue("end_time"))
	maxAmount := string(ctx.FormValue("max_amount"))
	minAmount := string(ctx.FormValue("min_amount"))
	state := string(ctx.FormValue("state"))

	page, err := strconv.ParseUint(string(ctx.FormValue("page")), 10, 64)
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.ParseUint(string(ctx.FormValue("page_size")), 10, 64)
	if err != nil {
		pageSize = 15
	}

	if startTime == "" || endTime == "" {
		helper.Print(ctx, false, helper.DateTimeErr)
		return
	}

	ex := make(map[string]interface{})
	rangeParam := map[string][]interface{}{}
	if minAmount != "" && maxAmount != "" {
		if !validator.CheckStringDigit(minAmount) || !validator.CheckStringDigit(maxAmount) {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}

		minAmountInt, _ := strconv.Atoi(minAmount)
		maxAmountInt, _ := strconv.Atoi(maxAmount)
		if minAmountInt > maxAmountInt {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}

		rangeParam["amount"] = []interface{}{minAmountInt, maxAmountInt}
	}

	if username != "" {
		if !validator.CheckUName(username, 4, 9) {
			helper.Print(ctx, false, helper.UsernameErr)
			return
		}

		ex["username"] = username
	}

	baseState := []interface{}{
		model.WithdrawSuccess,
		model.WithdrawFailed,
		model.WithdrawAbnormal,
	}
	ex["state"] = baseState

	if state != "" {
		stateInt, err := strconv.Atoi(state)
		if err != nil {
			helper.Print(ctx, false, helper.StateParamErr)
			return
		}

		if stateInt < model.WithdrawSuccess && stateInt > model.WithdrawAbnormal {
			helper.Print(ctx, false, helper.StateParamErr)
			return
		}

		ex["state"] = state
	}

	if id != "" {
		if !validator.CheckStringDigit(id) {
			helper.Print(ctx, false, helper.IDErr)
			return
		}

		ex = map[string]interface{}{
			"_id":   id,
			"state": baseState,
		}
	}

	data, err := model.WithdrawHistoryList(ex, rangeParam, 1, startTime, endTime, uint(page), uint(pageSize))
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	result, err := model.WithdrawDealListData(data)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, result)
}

// MemberWithdrawList 会员列表-提款信息
func (that *WithdrawController) MemberWithdrawList(ctx *fasthttp.RequestCtx) {

	financeType := string(ctx.FormValue("finance_type"))
	uid := string(ctx.FormValue("uid"))
	ty := string(ctx.FormValue("ty"))
	state := string(ctx.FormValue("state"))
	startTime := string(ctx.FormValue("start_time"))
	endTime := string(ctx.FormValue("end_time"))

	page, err := strconv.ParseUint(string(ctx.FormValue("page")), 10, 64)
	if err != nil {
		page = 1
	}

	pageSize, err := strconv.ParseUint(string(ctx.FormValue("page_size")), 10, 64)
	if err != nil {
		pageSize = 15
	}

	if startTime == "" || endTime == "" {
		helper.Print(ctx, false, helper.DateTimeErr)
		return
	}

	ex := g.Ex{}
	if uid == "" || !validator.CheckStringDigit(uid) {
		helper.Print(ctx, false, helper.UIDErr)
		return
	}

	ex["uid"] = uid

	if financeType != "" {
		financeTypeInt, err := strconv.Atoi(financeType)
		if err != nil {
			helper.Print(ctx, false, helper.FinanceTypeErr)
			return
		}
		if financeTypeInt != model.TransactionWithDraw &&
			financeTypeInt != model.TransactionValetWithdraw &&
			financeTypeInt != model.TransactionAgencyWithdraw {
			helper.Print(ctx, false, helper.FinanceTypeErr)
			return
		}

		ex["finance_type"] = financeTypeInt
	}

	if ty != "" {
		if !validator.CheckIntScope(ty, 1, 2) {
			helper.Print(ctx, false, helper.TimeTypeErr)
			return
		}
	}

	if state != "" {
		if !validator.CheckStringDigit(state) ||
			!validator.CheckIntScope(state, model.WithdrawReviewing, model.WithdrawDispatched) {
			helper.Print(ctx, false, helper.StateParamErr)
			return
		}
		ex["state"] = state
	}

	tyUint, err := strconv.ParseUint(ty, 10, 8)
	if err != nil {
		helper.Print(ctx, false, helper.TimeTypeErr)
		return
	}

	data, err := model.WithdrawList(ex, uint8(tyUint), startTime, endTime, uint(page), uint(pageSize))
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	result, err := model.WithdrawDealListData(data)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, result)
}

// FinanceReviewList 提款管理-财务审核
func (that *WithdrawController) FinanceReviewList(ctx *fasthttp.RequestCtx) {
	// 订单号
	id := string(ctx.FormValue("id"))
	startTime := string(ctx.FormValue("start_time"))
	endTime := string(ctx.FormValue("end_time"))
	maxAmount := string(ctx.FormValue("max_amount"))
	minAmount := string(ctx.FormValue("min_amount"))
	//会员账号
	username := string(ctx.FormValue("username"))
	//风控审核人
	confirmName := string(ctx.FormValue("confirm_name"))
	page, err := strconv.ParseUint(string(ctx.FormValue("page")), 10, 64)
	if err != nil {
		page = 1
	}

	pageSize, err := strconv.ParseUint(string(ctx.FormValue("page_size")), 10, 64)
	if err != nil {
		pageSize = 15
	}

	if startTime == "" || endTime == "" {
		helper.Print(ctx, false, helper.DateTimeErr)
		return
	}

	ex := g.Ex{}
	if minAmount != "" && maxAmount != "" {
		if !validator.CheckStringDigit(minAmount) || !validator.CheckStringDigit(maxAmount) {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}

		minAmountInt, _ := strconv.Atoi(minAmount)
		maxAmountInt, _ := strconv.Atoi(maxAmount)
		if minAmountInt > maxAmountInt {
			helper.Print(ctx, false, helper.AmountErr)
			return
		}

		ex["amount"] = g.Op{"between": exp.NewRangeVal(minAmountInt, maxAmountInt)}
	}

	if username != "" {
		if !validator.CheckUName(username, 4, 9) {
			helper.Print(ctx, false, helper.UsernameErr)
			return
		}

		ex["username"] = username
	}

	if confirmName != "" {
		if !validator.CheckAName(confirmName, 5, 20) {
			helper.Print(ctx, false, helper.AdminNameErr)
			return
		}

		ex["confirm_name"] = confirmName
	}

	if id != "" {
		if !validator.CheckStringDigit(id) {
			helper.Print(ctx, false, helper.IDErr)
			return
		}

		ex = map[string]interface{}{
			"id": id,
		}
	}

	ex["state"] = []int{
		model.WithdrawDealing,
		model.WithdrawAutoPayFailed,
	}

	data, err := model.WithdrawList(ex, 1, startTime, endTime, uint(page), uint(pageSize))
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	result, err := model.WithdrawDealListData(data)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, result)
}

// ReviewReject 后台审核拒绝
func (that *WithdrawController) ReviewReject(ctx *fasthttp.RequestCtx) {

	param := withdrawReviewReject{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	// 加锁
	err = model.WithdrawLock(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}
	defer model.WithdrawUnLock(param.ID)

	withdraw, err := model.WithdrawFind(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	// 出款分为自动出款和手动出款
	// 如果是自动出款, 状态必须是自动出款失败的才可以手动代付或出款
	// 如果是手动出款, 状态必须是处理中的才可以手动代付或出款
	if withdraw.Automatic == 1 {
		if withdraw.State != model.WithdrawAutoPayFailed {
			helper.Print(ctx, false, helper.OrderStateErr)
			return
		}
	} else {
		if withdraw.State != model.WithdrawDealing {
			helper.Print(ctx, false, helper.OrderStateErr)
			return
		}
	}

	admin, err := model.AdminToken(ctx)
	if err != nil {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	/*
		logMsg := fmt.Sprintf("拒绝【订单号:%s；会员账号:%s；订单金额:%.4f；申请时间:%s；完成时间:%s】",
			withdraw.ID, withdraw.Username, withdraw.Amount, model.TimeFormat(withdraw.CreatedAt), model.TimeFormat(ctx.Time().Unix()))
		defer model.SystemLogWrite(logMsg, ctx)
	*/

	record := g.Record{
		"review_remark":   param.Remark,
		"withdraw_remark": param.WithdrawRemark,
		"withdraw_at":     ctx.Time().Unix(),
		"withdraw_uid":    admin["id"],
		"withdraw_name":   admin["name"],
		"state":           model.WithdrawFailed,
	}

	err = model.WithdrawReject(param.ID, record)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// Review 审核
func (that *WithdrawController) Review(ctx *fasthttp.RequestCtx) {

	param := withdrawReviewParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	// 加锁
	err = model.WithdrawLock(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}
	defer model.WithdrawUnLock(param.ID)

	withdraw, err := model.WithdrawFind(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	// 出款分为自动出款和手动出款
	// 如果是自动出款, 状态必须是自动出款失败的才可以手动代付或出款
	// 如果是手动出款, 状态必须是处理中的才可以手动代付或出款
	if withdraw.Automatic == 1 {
		if withdraw.State != model.WithdrawAutoPayFailed {
			helper.Print(ctx, false, helper.OrderStateErr)
			return
		}
	} else {
		if withdraw.State != model.WithdrawDealing {
			helper.Print(ctx, false, helper.OrderStateErr)
			return
		}
	}

	admin, err := model.AdminToken(ctx)
	if err != nil {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	record := g.Record{
		"withdraw_remark": param.Remark,
		"withdraw_at":     ctx.Time().Unix(),
		"withdraw_uid":    admin["id"],
		"withdraw_name":   admin["name"],
	}

	if param.Ty == 1 { // 三方代付
		err = model.WithdrawHandToAuto(withdraw.UID, withdraw.Username, withdraw.ID, param.Pid, withdraw.BID, withdraw.Amount, ctx.Time())
		if err != nil {
			helper.Print(ctx, false, err.Error())
			return
		}
	}

	if param.Ty == 2 { // 人工出款

		//if !validator.CheckStringDigit(param.CardNo) || !validator.CheckStringLength(param.CardNo, 6, 20) {
		//	helper.Print(ctx, false, helper.BankcardIDErr)
		//	return
		//}

		/*
			logMsg := fmt.Sprintf("人工出款【订单号:%s；会员账号:%s；订单金额:%.4f；申请时间:%s；完成时间:%s】",
				withdraw.ID, withdraw.Username, withdraw.Amount, model.TimeFormat(withdraw.CreatedAt), model.TimeFormat(ctx.Time().Unix()))
			defer model.SystemLogWrite(logMsg, ctx)
		*/

		record["pid"] = "0"
		//record["bid"] = param.BankId
		record["automatic"] = "0"
		record["state"] = model.WithdrawSuccess
		record["bank_name"] = param.BankName
		record["card_no"] = param.CardNo
		record["real_name"] = param.RealName

		bk, err := model.BankCardByCol(param.CardNo)
		if err != nil && err != sql.ErrNoRows {
			helper.Print(ctx, false, err.Error())
			return
		}

		if err == nil {
			//提款超过当日最大提款限额
			fishAmount, _ := decimal.NewFromString(bk.DailyFinishAmount)
			maxAmount, _ := decimal.NewFromString(bk.DailyMaxAmount)
			totalAmount, _ := decimal.NewFromString(bk.TotalFinishAmount)
			totalMaxAmount, _ := decimal.NewFromString(bk.TotalMaxAmount)

			if fishAmount.Cmp(maxAmount) >= 0 || fishAmount.Add(decimal.NewFromFloat(withdraw.Amount)).GreaterThan(maxAmount) ||
				totalAmount.Add(decimal.NewFromFloat(withdraw.Amount)).GreaterThan(totalMaxAmount) {
				helper.Print(ctx, false, helper.DailyAmountLimitErr)
				return
			}
			err = model.WithdrawHandSuccess(param.ID, withdraw.UID, withdraw.BID, record)
			if err != nil {
				helper.Print(ctx, false, err.Error())
				return
			}
			record := g.Record{
				"daily_finish_amount": fishAmount.Add(decimal.NewFromFloat(withdraw.Amount)).StringFixed(4),
				"total_finish_amount": totalAmount.Add(decimal.NewFromFloat(withdraw.Amount)).StringFixed(4),
			}
			if fishAmount.Add(decimal.NewFromFloat(withdraw.Amount)).Cmp(maxAmount) >= 0 {
				record["state"] = 0
			}
			if totalAmount.Add(decimal.NewFromFloat(withdraw.Amount)).Cmp(totalMaxAmount) >= 0 {
				record["state"] = 0
			}
			model.BankCardUpdate(bk.Id, record)
		} else {
			err = model.WithdrawHandSuccess(param.ID, withdraw.UID, withdraw.BID, record)
			if err != nil {
				helper.Print(ctx, false, err.Error())
				return
			}
		}

	}

	helper.Print(ctx, true, helper.Success)

}

// AutomaticFailed 代付失败
func (that *WithdrawController) AutomaticFailed(ctx *fasthttp.RequestCtx) {

	id := string(ctx.FormValue("id"))
	if !validator.CheckStringDigit(id) {
		helper.Print(ctx, false, helper.IDErr)
		return
	}

	// 加锁
	err := model.WithdrawLock(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}
	defer model.WithdrawUnLock(id)

	admin, err := model.AdminToken(ctx)
	if err != nil {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	err = model.WithdrawAutoPaySetFailed(id, ctx.Time().Unix(), admin["id"], admin["name"])
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// Limit 每日剩余提款次数和总额
func (that *WithdrawController) Limit(ctx *fasthttp.RequestCtx) {

	data, err := model.WithdrawLimit(ctx)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// Processing 获取正在处理中的提现订单
func (that *WithdrawController) Processing(ctx *fasthttp.RequestCtx) {

	data, err := model.WithdrawInProcessing(ctx)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, data)
}

// RiskReview 风控审核 通过
func (that *WithdrawController) RiskReview(ctx *fasthttp.RequestCtx) {

	id := string(ctx.FormValue("id"))
	if !validator.CheckStringDigit(id) {
		helper.Print(ctx, false, helper.IDErr)
		return
	}

	// 加锁
	err := model.WithdrawLock(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}
	defer model.WithdrawUnLock(id)

	withdraw, err := model.WithdrawFind(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if withdraw.State != model.WithdrawDispatched {
		helper.Print(ctx, false, helper.OrderStateErr)
		return
	}

	record := g.Record{
		"state":      model.WithdrawDealing,
		"confirm_at": ctx.Time().Unix(),
	}

	err = model.WithdrawUpdateInfo(id, record)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	_ = model.SetRisksOrder(withdraw.ConfirmUID, id, -1)

	if withdraw.Automatic == 1 {
		fmt.Println("调用第三方代付")

		bankcardNo, realName, err := model.WithdrawGetBkAndRn(withdraw.BID, withdraw.UID, false)
		if err != nil {
			record = g.Record{
				"state":     model.WithdrawAutoPayFailed,
				"automatic": "1",
			}
			_ = model.WithdrawUpdateInfo(id, record)
			helper.Print(ctx, false, err.Error())
			return
		}

		bankcard, err := model.WithdrawGetBank(withdraw.BID, withdraw.Username)
		if err != nil {
			record = g.Record{
				"state":     model.WithdrawAutoPayFailed,
				"automatic": "1",
			}
			_ = model.WithdrawUpdateInfo(id, record)
			helper.Print(ctx, false, err.Error())
			return
		}

		p := model.WithdrawAutoParam{
			OrderID:     withdraw.ID,
			Amount:      decimal.NewFromFloat(withdraw.Amount).Mul(decimal.NewFromInt(1000)).String(),
			BankID:      bankcard.BankID,
			CardNumber:  bankcardNo, // 银行卡号
			CardName:    realName,   // 持卡人姓名
			Ts:          ctx.Time(), // 时间
			BankAddress: bankcard.BankAddress,
		}

		// 自动出款的错误不返回给前端, mysql修改成功了就是成功了
		err = model.WithdrawAuto(p, withdraw.Level)
		if err != nil {
			record = g.Record{
				"state":     model.WithdrawAutoPayFailed,
				"automatic": "1",
			}
			err = model.WithdrawUpdateInfo(id, record)
			if err != nil {
				helper.Print(ctx, false, err.Error())
				return
			}
		}
	}

	helper.Print(ctx, true, helper.Success)
}

// RiskReviewReject 风控审核 拒绝
func (that *WithdrawController) RiskReviewReject(ctx *fasthttp.RequestCtx) {

	param := withdrawRiskRejectParam{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	// 加锁
	err = model.WithdrawLock(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}
	defer model.WithdrawUnLock(param.ID)

	withdraw, err := model.WithdrawFind(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if withdraw.State != model.WithdrawDispatched {
		helper.Print(ctx, false, helper.OrderStateErr)
		return
	}

	record := g.Record{
		"state":           model.WithdrawReviewReject,
		"review_remark":   param.ReviewRemark,
		"withdraw_remark": param.WithdrawRemark,
		"confirm_at":      ctx.Time().Unix(),
	}
	err = model.WithdrawRiskReview(param.ID, model.WithdrawReviewReject, record, withdraw)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, true, helper.Success)
}

// HangUp 订单挂起
func (that *WithdrawController) HangUp(ctx *fasthttp.RequestCtx) {

	param := withdrawHangUp{}
	err := validator.Bind(ctx, &param)
	if err != nil {
		helper.Print(ctx, false, helper.ParamErr)
		return
	}

	// 加锁
	err = model.WithdrawLock(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}
	defer model.WithdrawUnLock(param.ID)

	withdraw, err := model.WithdrawFind(param.ID)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if withdraw.State != model.WithdrawDispatched {
		helper.Print(ctx, false, helper.OrderStateErr)
		return
	}

	admin, err := model.AdminToken(ctx)
	if err != nil {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	if withdraw.ConfirmUID != admin["id"] {
		helper.Print(ctx, false, helper.MethodNoPermission)
		return
	}

	record := g.Record{
		"hang_up_uid":    admin["id"],
		"hang_up_remark": param.HangUpRemark,
		"hang_up_name":   admin["name"],
		"remark_id":      param.RemarkID,
		"state":          model.WithdrawHangup,
		"hang_up_at":     ctx.Time().Unix(),
		"receive_at":     "0",
	}

	err = model.WithdrawUpdateInfo(param.ID, record)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	_ = model.SetRisksOrder(withdraw.ConfirmUID, param.ID, -1)

	helper.Print(ctx, true, helper.Success)
}

// ConfirmNameUpdate 修改领取人
func (that *WithdrawController) ConfirmNameUpdate(ctx *fasthttp.RequestCtx) {

	id := string(ctx.FormValue("id"))
	if !validator.CheckStringDigit(id) {
		helper.Print(ctx, false, helper.IDErr)
		return
	}

	// 加锁
	err := model.WithdrawLock(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}
	defer model.WithdrawUnLock(id)

	withdraw, err := model.WithdrawFind(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if withdraw.State != model.WithdrawDispatched {
		helper.Print(ctx, false, helper.OrderStateErr)
		return
	}

	newConfirmUID := string(ctx.FormValue("confirm_uid"))
	newConfirmName := string(ctx.FormValue("confirm_name"))
	if newConfirmName == "" || newConfirmUID == "" {
		helper.Print(ctx, false, helper.AdminNameErr)
		return
	}

	record := g.Record{
		"confirm_uid":  newConfirmUID,
		"confirm_name": newConfirmName,
		"receive_at":   ctx.Time().Unix(),
	}
	err = model.WithdrawUpdateInfo(id, record)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	_ = model.SetRisksOrder(withdraw.ConfirmUID, id, -1)
	_ = model.SetRisksOrder(newConfirmUID, id, 1)

	helper.Print(ctx, true, helper.Success)
}

// ConfirmName 领取
func (that *WithdrawController) ConfirmName(ctx *fasthttp.RequestCtx) {

	id := string(ctx.FormValue("id"))
	if !validator.CheckStringDigit(id) {
		helper.Print(ctx, false, helper.IDErr)
		return
	}

	// 加锁
	err := model.WithdrawLock(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}
	defer model.WithdrawUnLock(id)

	withdraw, err := model.WithdrawFind(id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	if withdraw.State != model.WithdrawHangup && withdraw.State != model.WithdrawReviewing {
		helper.Print(ctx, false, helper.OrderTakenAlready)
		return
	}

	admin, err := model.AdminToken(ctx)
	if err != nil {
		helper.Print(ctx, false, helper.AccessTokenExpires)
		return
	}

	record := g.Record{
		"confirm_uid":  admin["id"],
		"confirm_name": admin["name"],
		"state":        model.WithdrawDispatched,
		"receive_at":   ctx.Time().Unix(),
	}
	err = model.WithdrawUpdateInfo(id, record)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	_ = model.SetRisksOrder(admin["id"], id, 1)

	helper.Print(ctx, true, helper.Success)
}

// BankCardWithdrawRecord 获取银行卡 绑定时间 成功失败次数
func (that *WithdrawController) BankCardWithdrawRecord(ctx *fasthttp.RequestCtx) {

	bankID := string(ctx.FormValue("bid"))
	username := string(ctx.FormValue("username"))

	if bankID == "" || !validator.CheckStringDigit(bankID) {
		helper.Print(ctx, false, helper.BankcardIDErr)
		return
	}

	if username == "" || !validator.CheckUName(username, 4, 9) {
		helper.Print(ctx, false, helper.UsernameErr)
		return
	}

	bankcard, err := model.WithdrawGetBank(bankID, username)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	// 获取银行卡成功失败的次数
	success, fail := model.WithdrawBanKCardNumber(bankID)

	data := withdrawRecord{
		Success: success,
		Fail:    fail,
		BindAt:  bankcard.CreatedAt,
	}

	helper.Print(ctx, true, data)
}
