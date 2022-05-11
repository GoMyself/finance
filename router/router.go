package router

import (
	"fmt"
	"runtime/debug"
	"time"

	"finance/controller"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

var (
	ApiTimeoutMsg = `{"status": "false","data":"服务器响应超时，请稍后重试"}`
	ApiTimeout    = time.Second * 30
	route         *router.Router
	buildInfo     BuildInfo
)

type BuildInfo struct {
	GitReversion   string
	BuildTime      string
	BuildGoVersion string
}

func apiServerPanic(ctx *fasthttp.RequestCtx, rcv interface{}) {

	err := rcv.(error)
	fmt.Println(err)
	debug.PrintStack()

	if r := recover(); r != nil {
		fmt.Println("recovered failed", r)
	}

	ctx.SetStatusCode(500)
	return
}

func Version(ctx *fasthttp.RequestCtx) {

	ctx.SetContentType("text/html; charset=utf-8")
	fmt.Fprintf(ctx, "sms & email<br />Git reversion = %s<br />Build Time = %s<br />Go version = %s<br />System Time = %s<br />",
		buildInfo.GitReversion, buildInfo.BuildTime, buildInfo.BuildGoVersion, ctx.Time())

	// ctx.Request.Header.VisitAll(func (key, value []byte) {
	//	fmt.Fprintf(ctx, "%s: %s<br/>", string(key), string(value))
	// })
}

// SetupRouter 设置路由列表
func SetupRouter(b BuildInfo) *router.Router {

	route = router.New()
	route.PanicHandler = apiServerPanic

	buildInfo = b

	payCtl := new(controller.PayController)
	cbCtl := new(controller.CallBackController)
	wdCtl := new(controller.WithdrawController)
	cateCtl := new(controller.CateController)
	channelCtl := new(controller.ChannelController)
	vipCtl := new(controller.VipController)
	depositCtl := new(controller.DepositController)
	bankCtl := new(controller.ChannelBankController)
	tunnelCtl := new(controller.TunnelController)
	promoCtl := new(controller.PromoController)
	risksCtl := new(controller.RisksController)
	creditCtl := new(controller.CreditLevelController)
	lockCtl := new(controller.LockController)
	usdtCtl := new(controller.UsdtController)
	bankCardCtl := new(controller.BankCardController)
	manualCtl := new(controller.ManualController)

	route_callback_group := route.Group("/finance/callback")
	route_merchant_group := route.Group("/merchant/finance")

	// [callback] uz pay 代收 回调
	post(route_callback_group, "/uzd", cbCtl.UZD)
	// [callback] uz pay 代付 回调
	post(route_callback_group, "/uzw", cbCtl.UZW)
	// [callback] w pay 代收回调
	post(route_callback_group, "/wd", cbCtl.WD)
	// [callback] w pay 代付回调
	post(route_callback_group, "/ww", cbCtl.WW)
	// [callback] 优付宝 pay 代收回调
	post(route_callback_group, "/yfbd", cbCtl.YFBD)
	// [callback] 优付宝 pay 代付回调
	post(route_callback_group, "/yfbw", cbCtl.YFBW)
	// [callback] 风杨 pay 代收回调
	post(route_callback_group, "/fyd", cbCtl.FYD)
	// [callback] 风杨 pay 代付回调
	post(route_callback_group, "/fyw", cbCtl.FYW)
	// [callback] quick pay 代收回调
	post(route_callback_group, "/quickd", cbCtl.QuickD)
	// [callback] quick pay 代付回调
	post(route_callback_group, "/quickw", cbCtl.QuickW)
	// [callback] USDT 代收回调
	get(route_callback_group, "/usdtd", cbCtl.UsdtD)
	// [callback] 越南支付代收回调
	post(route_callback_group, "/ynd", cbCtl.YND)
	// [callback] 越南支付代收回调
	post(route_callback_group, "/ynw", cbCtl.YNW)
	// [callback] vt pay 代收回调
	post(route_callback_group, "/vtd", cbCtl.VTD)
	// [callback] vt pay 代付回调
	post(route_callback_group, "/vtw", cbCtl.VTW)
	// [callback] 918 pay 代收回调
	post(route_callback_group, "/jybtd", cbCtl.JYBD)
	// [callback] 918 pay 代付回调
	post(route_callback_group, "/jybtw", cbCtl.JYBW)
	// [callback] 918 pay 代收回调
	post(route_callback_group, "/vnd", cbCtl.VND)
	// [callback] 918 pay 代付回调
	post(route_callback_group, "/vnw", cbCtl.VNW)
	// [callback] 918 pay 代收回调
	post(route_callback_group, "/dbd", cbCtl.DBD)
	// [callback] 918 pay 代付回调
	post(route_callback_group, "/dbw", cbCtl.DBW)

	// [前台] 存款渠道
	get(nil, "/finance/cate", payCtl.Cate)
	// [前台] 存款通道
	get(nil, "/finance/tunnel", payCtl.Tunnel)
	// [前台] 发起存款
	post(nil, "/finance/pay", payCtl.Pay)
	// [前台] 用户申请提现
	post(nil, "/finance/withdraw", wdCtl.Withdraw)
	// [前台] 用户提现剩余次数和额度
	get(nil, "/finance/withdraw/limit", wdCtl.Limit)
	// [前台] 获取正在处理中的提现订单
	get(nil, "/finance/withdraw/processing", wdCtl.Processing)
	// [前台] 渠道列表数据缓存
	get(nil, "/finance/cate/cache", cateCtl.Cache)
	// [前台] 通道列表数据缓存
	//get(nil, "/finance/channel/cache", channelCtl.Cache)
	// [前台] 线下转卡-发起存款
	post(nil, "/finance/manual/pay", manualCtl.Pay)
	// [前台] 线下转卡-银行卡列表
	post(nil, "/finance/bankcard/list", bankCardCtl.BankCards)
	// [前台] 线下USDT-发起存款
	post(nil, "/finance/usdt/pay", usdtCtl.Pay)

	// [前台] 线下USDT-获取trc收款地址
	get(nil, "/finance/usdt/info", usdtCtl.Info)

	// [商户后台] 财务管理-渠道管理-通道优惠管理-通道优惠存款
	get(route_merchant_group, "/promo/detail", promoCtl.Detail)
	// [商户后台] 财务管理-渠道管理-通道优惠管理-开启/关闭优惠状态
	get(route_merchant_group, "/promo/update/state", promoCtl.UpdateState)
	// [商户后台] 财务管理-渠道管理-通道优惠管理-设置通道优惠比例
	post(route_merchant_group, "/promo/update/quota", promoCtl.UpdateQuota)

	// [商户后台] 财务管理-渠道管理-新增
	post(route_merchant_group, "/cate/insert", cateCtl.Insert)
	// [商户后台] 财务管理-渠道管理-修改
	post(route_merchant_group, "/cate/update", cateCtl.Update)
	// [商户后台] 财务管理-渠道管理-删除
	// post("/cate/delete", cateCtl.Delete)
	// [商户后台] 财务管理-渠道管理-列表
	post(route_merchant_group, "/cate/list", cateCtl.List)
	// [商户后台] 财务管理-渠道管理-启用/停用
	post(route_merchant_group, "/cate/update/state", cateCtl.UpdateState)
	// [商户后台] 渠道列表数据缓存
	get(route_merchant_group, "/cate/cache", cateCtl.Cache)
	// [商户后台] 财务管理-提款渠道
	post(route_merchant_group, "/cate/withdraw", cateCtl.Withdraw)

	// [商户后台] 财务管理-渠道管理-通道管理-新增
	post(route_merchant_group, "/channel/insert", channelCtl.Insert)
	// [商户后台] 财务管理-渠道管理-通道管理-修改
	post(route_merchant_group, "/channel/update", channelCtl.Update)
	// [商户后台] 财务管理-渠道管理-通道管理-删除
	// post("/channel/delete", channelCtl.Delete)
	// [商户后台] 财务管理-渠道管理-通道管理-列表
	get(route_merchant_group, "/channel/list", channelCtl.List)
	// [商户后台] 通道列表数据缓存
	//get(route_merchant_group, "/channel/cache", channelCtl.Cache)
	// [商户后台] 财务管理-渠道管理-通道管理-启用/停用
	post(route_merchant_group, "/channel/update/state", channelCtl.UpdateState)

	// [商户后台] 财务管理-渠道管理-会员等级通道-新增
	post(route_merchant_group, "/vip/insert", vipCtl.Insert)
	// [商户后台] 财务管理-渠道管理-会员等级通道-修改
	// [商户后台] 财务管理-渠道管理-会员等级通道-列表
	post(route_merchant_group, "/vip/update", vipCtl.Update)
	// [商户后台] 财务管理-渠道管理-会员等级通道-删除
	// [商户后台] 财务管理-渠道管理-会员等级通道-删除
	post(route_merchant_group, "/vip/delete", vipCtl.Delete)
	// [商户后台] 财务管理-渠道管理-会员等级通道-列表
	get(route_merchant_group, "/vip/list", vipCtl.List)
	// [商户后台] 财务管理-渠道管理-会员等级通道-启用/停用
	post(route_merchant_group, "/vip/update/state", vipCtl.UpdateState)

	// [商户后台] 财务管理-渠道管理-通道银行管理-新增
	post(route_merchant_group, "/bank/insert", bankCtl.Insert)
	// [商户后台] 财务管理-渠道管理-通道银行管理-修改
	post(route_merchant_group, "/bank/update", bankCtl.Update)
	// [商户后台] 财务管理-渠道管理-通道银行管理-列表
	post(route_merchant_group, "/bank/list", bankCtl.List)
	// [商户后台] 财务管理-渠道管理-通道银行管理-启用/停用
	post(route_merchant_group, "/bank/update/state", bankCtl.UpdateState)

	// [商户后台] 财务管理-渠道管理-会员信用等级-新增
	post(route_merchant_group, "/credit/insert", creditCtl.Insert)
	// [商户后台] 财务管理-渠道管理-会员信用等级-修改
	post(route_merchant_group, "/credit/update", creditCtl.Update)
	// [商户后台] 财务管理-渠道管理-会员信用等级-列表
	post(route_merchant_group, "/credit/list", creditCtl.List)
	// [商户后台] 财务管理-渠道管理-会员信用等级-启用/停用
	post(route_merchant_group, "/credit/update/state", creditCtl.UpdateState)

	// [商户后台] 财务管理-渠道管理-会员信用等级-新增会员
	post(route_merchant_group, "/membercredit/insert", creditCtl.MemberInsert)
	// [商户后台] 财务管理-渠道管理-会员信用等级-列表会员
	post(route_merchant_group, "/membercredit/list", creditCtl.MemberList)
	// [商户后台] 财务管理-渠道管理-会员信用等级-删除会员
	post(route_merchant_group, "/membercredit/delete", creditCtl.MemberDelete)

	// [商户后台] 财务管理-渠道管理-会员锁定-新增
	post(route_merchant_group, "/memberlock/insert", lockCtl.MemberInsert)
	// [商户后台] 财务管理-渠道管理-会员锁定-列表
	post(route_merchant_group, "/memberlock/list", lockCtl.MemberList)
	// [商户后台] 财务管理-渠道管理-会员锁定-启用
	post(route_merchant_group, "/memberlock/update/state", lockCtl.UpdateState)

	// [商户后台] 财务管理-渠道管理-通道类型管理-列表
	get(route_merchant_group, "/tunnel/list", tunnelCtl.List)
	// [商户后台] 财务管理-渠道管理-通道类型管理-修改
	post(route_merchant_group, "/tunnel/update", tunnelCtl.Update)
	// [商户后台] 财务管理-提款管理-会员列表-提款
	post(route_merchant_group, "/withdraw/memberlist", wdCtl.MemberWithdrawList)
	// [商户后台] 财务管理-提款管理-提款列表
	post(route_merchant_group, "/withdraw/financelist", wdCtl.FinanceReviewList)
	// [商户后台] 财务管理-提款管理-提款历史记录
	post(route_merchant_group, "/withdraw/historylist", wdCtl.HistoryList)
	// [商户后台] 财务管理-提款管理-拒绝
	post(route_merchant_group, "/withdraw/reject", wdCtl.ReviewReject)
	// [商户后台] 财务管理-提款管理-人工出款（手动代付， 手动出款）
	post(route_merchant_group, "/withdraw/review", wdCtl.Review)
	// [商户后台] 财务管理-提款管理-代付失败
	post(route_merchant_group, "/withdraw/automatic/failed", wdCtl.AutomaticFailed)

	// [商户后台] 风控管理-提款审核-待领取列表
	post(route_merchant_group, "/withdraw/waitreceive", wdCtl.RiskWaitConfirmList)
	// [商户后台] 风控管理-提款审核-待领取列表-银行卡交易记录统计
	post(route_merchant_group, "/withdraw/cardrecord", wdCtl.BankCardWithdrawRecord)
	// [商户后台] 风控管理-提款审核-待审核列表
	post(route_merchant_group, "/withdraw/waitreview", wdCtl.RiskReviewList)
	// [商户后台] 风控管理-提款审核-待审核列表-通过
	post(route_merchant_group, "/withdraw/reviewpass", wdCtl.RiskReview)
	// [商户后台] 风控管理-提款审核-待审核列表-拒绝
	post(route_merchant_group, "/withdraw/reviewreject", wdCtl.RiskReviewReject)
	// [商户后台] 风控管理-提款审核-待审核列表-挂起
	post(route_merchant_group, "/withdraw/hangup", wdCtl.HangUp)
	// [商户后台] 风控管理-提款审核-挂起列表
	post(route_merchant_group, "/withdraw/hanguplist", wdCtl.HangUpList)
	// [商户后台] 风控管理-提款审核-待审核列表-修改领取人
	post(route_merchant_group, "/withdraw/receiveupdate", wdCtl.ConfirmNameUpdate)
	// [商户后台] 风控管理-提款审核-挂起列表-领取
	post(route_merchant_group, "/withdraw/receive", wdCtl.ConfirmName)
	// [商户后台] 风控管理-提款审核-历史记录列表
	post(route_merchant_group, "/withdraw/riskhistory", wdCtl.RiskHistory)

	// [商户后台] 会员列表-存款管理-存款信息
	get(route_merchant_group, "/deposit/detail", depositCtl.Detail)
	// [商户后台] 财务管理-存款管理-入款订单列表/补单审核列表
	get(route_merchant_group, "/deposit/list", depositCtl.List)
	// [商户后台] 财务管理-存款管理-历史记录
	get(route_merchant_group, "/deposit/history", depositCtl.History)
	// [商户后台] 财务管理-存款管理-存款补单
	post(route_merchant_group, "/deposit/manual", depositCtl.Manual)
	// [商户后台] 财务管理-存款管理-补单审核
	post(route_merchant_group, "/deposit/review", depositCtl.Review)
	// [商户后台] 财务管理-手动下分
	post(route_merchant_group, "/deposit/reduce", depositCtl.Reduce)
	// [商户后台] 财务管理-存款管理-USDT存款
	post(route_merchant_group, "/deposit/usdt/list", depositCtl.USDTList)
	// [商户后台] 财务管理-存款管理-线下转卡-入款订单
	post(route_merchant_group, "/deposit/manual/list", depositCtl.Offline)
	// [商户后台] 财务管理-存款管理-线下转卡-确认金额待审核
	post(route_merchant_group, "/deposit/manual/reviewing", depositCtl.OfflineToReview)
	// [商户后台] 财务管理-存款管理-线下转卡-审核
	post(route_merchant_group, "/deposit/manual/review", depositCtl.OfflineReview)
	// [商户后台] 财务管理-存款管理-线下USDT-确认金额待审核
	post(route_merchant_group, "/deposit/usdt/reviewing", depositCtl.OfflineUSDT)
	// [商户后台] 财务管理-存款管理-线下USDT-审核
	post(route_merchant_group, "/deposit/usdt/review", depositCtl.OfflineUSDTReview)

	// [商户后台] 财务管理-存款管理-线下转卡-添加银行卡
	post(route_merchant_group, "/bankcard/insert", bankCardCtl.Insert)
	// [商户后台] 财务管理-存款管理-线下转卡-列表银行卡
	post(route_merchant_group, "/bankcard/list", bankCardCtl.List)
	// [商户后台] 财务管理-存款管理-线下转卡-更新银行卡
	post(route_merchant_group, "/bankcard/update", bankCardCtl.Update)
	// [商户后台] 财务管理-存款管理-线下转卡-删除银行卡
	get(route_merchant_group, "/bankcard/delete", bankCardCtl.Delete)

	/*
		// [商户后台] 财务管理-渠道管理-usdt汇率设置
		post(route_merchant_group, "/usdt/setrate", usdtCtl.SetRate)
		// [商户后台] 财务管理-渠道管理-usdt汇率获取
		get(route_merchant_group, "/usdt/getrate", usdtCtl.GetRate)
		// [商户后台] 财务管理-存款管理-线下usdt-设置收款地址
		post(route_merchant_group, "/usdt/settrc", usdtCtl.SetTRC)
		// [商户后台] 财务管理-存款管理-线下usdt-获取收款地址
		get(route_merchant_group, "/usdt/gettrc", usdtCtl.GetTRC)
	*/

	get(route_merchant_group, "/usdt/info", usdtCtl.Info)
	post(route_merchant_group, "/usdt/update", usdtCtl.Update)

	// [商户后台] 风控管理-风控配置-接单控制-关闭自动派单
	get(route_merchant_group, "/risks/close", risksCtl.CloseAuto)
	// [商户后台] 风控管理-风控配置-接单控制-开启自动派单
	get(route_merchant_group, "/risks/open", risksCtl.OpenAuto)
	// [商户后台] 风控管理-风控配置-获取自动派单状态
	get(route_merchant_group, "/risks/state", risksCtl.State)
	// [商户后台] 风控管理-风控配置-获取自动派单人员的列表
	get(route_merchant_group, "/risks/list", risksCtl.List)
	// [商户后台] 风控管理-风控配置-设置接单数量
	get(route_merchant_group, "/risks/setnumer", risksCtl.SetNumber)
	// [商户后台] 风控管理-风控配置-领取人列表
	get(route_merchant_group, "/risks/receives", risksCtl.Receives)
	// [商户后台] 风控管理-风控配置-领取人数量
	get(route_merchant_group, "/risks/number", risksCtl.Number)
	// [商户后台] 风控管理-风控配置-设置同设备号注册数量
	post(route_merchant_group, "/risks/setregmax", risksCtl.SetRegMax)
	// [商户后台] 风控管理-风控配置-获取同设备号注册数量
	get(route_merchant_group, "/risks/regmax", risksCtl.RegMax)

	return route
}

// get is a shortcut for router.GET(path string, handle fasthttp.RequestHandler)
func get(g *router.Group, path string, handle fasthttp.RequestHandler) {

	if g != nil {
		g.GET(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
	} else {
		route.GET(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
	}

}

// head is a shortcut for router.HEAD(path string, handle fasthttp.RequestHandler)
func head(g *router.Group, path string, handle fasthttp.RequestHandler) {

	if g != nil {
		g.HEAD(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
	} else {
		route.HEAD(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
	}

}

// options is a shortcut for router.OPTIONS(path string, handle fasthttp.RequestHandler)
func options(g *router.Group, path string, handle fasthttp.RequestHandler) {

	if g != nil {
		g.OPTIONS(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
	} else {
		route.OPTIONS(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
	}

}

// post is a shortcut for router.POST(path string, handle fasthttp.RequestHandler)
func post(g *router.Group, path string, handle fasthttp.RequestHandler) {

	if g != nil {
		g.POST(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
	} else {
		route.POST(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
	}

}

// put is a shortcut for router.PUT(path string, handle fasthttp.RequestHandler)
func put(g *router.Group, path string, handle fasthttp.RequestHandler) {

	if g != nil {
		g.PUT(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
	} else {
		route.PUT(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
	}

}

// patch is a shortcut for router.PATCH(path string, handle fasthttp.RequestHandler)
func patch(g *router.Group, path string, handle fasthttp.RequestHandler) {

	if g != nil {
		g.PATCH(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
	} else {
		route.PATCH(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
	}

}

// delete is a shortcut for router.DELETE(path string, handle fasthttp.RequestHandler)
func delete(g *router.Group, path string, handle fasthttp.RequestHandler) {

	if g != nil {
		g.DELETE(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
	} else {
		route.DELETE(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
	}

}
