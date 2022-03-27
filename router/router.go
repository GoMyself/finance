package router

import (
	"fmt"
	"runtime/debug"
	"time"

	"finance/controller"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

var (
	ApiTimeoutMsg = `{"status": "false","data":"服务器响应超时，请稍后重试"}`
	ApiTimeout    = time.Second * 30
	router        *fasthttprouter.Router
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
func SetupRouter(b BuildInfo) *fasthttprouter.Router {

	router = fasthttprouter.New()
	router.PanicHandler = apiServerPanic

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

	// [callback] uz pay 代收 回调
	post("/finance/callback/uzd", cbCtl.UZD)
	// [callback] uz pay 代付 回调
	post("/finance/callback/uzw", cbCtl.UZW)
	// [callback] w pay 代收回调
	post("/finance/callback/wd", cbCtl.WD)
	// [callback] w pay 代付回调
	post("/finance/callback/ww", cbCtl.WW)
	// [callback] 优付宝 pay 代收回调
	post("/finance/callback/yfbd", cbCtl.YFBD)
	// [callback] 优付宝 pay 代付回调
	post("/finance/callback/yfbw", cbCtl.YFBW)
	// [callback] 风杨 pay 代收回调
	post("/finance/callback/fyd", cbCtl.FYD)
	// [callback] 风杨 pay 代付回调
	post("/finance/callback/fyw", cbCtl.FYW)
	// [callback] quick pay 代收回调
	post("/finance/callback/quickd", cbCtl.QuickD)
	// [callback] quick pay 代付回调
	post("/finance/callback/quickw", cbCtl.QuickW)
	// [callback] USDT 代收回调
	get("/finance/callback/usdtd", cbCtl.UsdtD)
	// [callback] 越南支付代收回调
	post("/finance/callback/ynd", cbCtl.YND)
	// [callback] 越南支付代收回调
	post("/finance/callback/ynw", cbCtl.YNW)

	// [前台] 存款渠道
	get("/finance/cate", payCtl.Cate)
	// [前台] 存款通道
	get("/finance/tunnel", payCtl.Tunnel)
	// [前台] 发起存款
	post("/finance/pay", payCtl.Pay)
	// [前台] 用户申请提现
	post("/finance/withdraw", wdCtl.Withdraw)
	// [前台] 用户提现剩余次数和额度
	get("/finance/withdraw/limit", wdCtl.Limit)
	// [前台] 获取正在处理中的提现订单
	get("/finance/withdraw/processing", wdCtl.Processing)
	// [前台] 渠道列表数据缓存
	get("/finance/cate/cache", cateCtl.Cache)
	// [前台] 通道列表数据缓存
	get("/finance/channel/cache", channelCtl.Cache)
	// [前台] usdt汇率
	get("/finance/usdt/rate", usdtCtl.GetRate)
	// [前台] 线下转卡-发起存款
	post("/finance/manual", payCtl.Manual)
	// [前台] 线下转卡-银行卡列表
	post("/finance/bankcard/list", bankCardCtl.BankCards)
	// [前台] 线下USDT-发起存款
	post("/finance/usdt", payCtl.USDT)
	// [前台] 线下USDT-获取trc收款地址
	get("/finance/usdt/address/trc", usdtCtl.GetTRC)

	// [商户后台] 财务管理-渠道管理-通道优惠管理-通道优惠存款
	get("/merchant/finance/promo/detail", promoCtl.Detail)
	// post("/merchant/finance/promo/update", promoCtl.Update)
	// [商户后台] 财务管理-渠道管理-通道优惠管理-开启/关闭优惠状态
	get("/merchant/finance/promo/updatestate", promoCtl.UpdateState)
	// [商户后台] 财务管理-渠道管理-通道优惠管理-设置通道优惠比例
	post("/merchant/finance/promo/updatequota", promoCtl.UpdateQuota)

	// [商户后台] 财务管理-渠道管理-新增
	post("/merchant/finance/cate/insert", cateCtl.Insert)
	// [商户后台] 财务管理-渠道管理-修改
	post("/merchant/finance/cate/update", cateCtl.Update)
	// [商户后台] 财务管理-渠道管理-删除
	// post("/merchant/finance/cate/delete", cateCtl.Delete)
	// [商户后台] 财务管理-渠道管理-列表
	post("/merchant/finance/cate/list", cateCtl.List)
	// [商户后台] 财务管理-渠道管理-启用/停用
	post("/merchant/finance/cate/updatestate", cateCtl.UpdateState)
	// [商户后台] 渠道列表数据缓存
	get("/merchant/finance/cate/cache", cateCtl.Cache)
	// [商户后台] 财务管理-提款渠道
	post("/merchant/finance/cate/withdraw", cateCtl.Withdraw)

	// [商户后台] 财务管理-渠道管理-通道管理-新增
	post("/merchant/finance/channel/insert", channelCtl.Insert)
	// [商户后台] 财务管理-渠道管理-通道管理-修改
	post("/merchant/finance/channel/update", channelCtl.Update)
	// [商户后台] 财务管理-渠道管理-通道管理-删除
	// post("/merchant/finance/channel/delete", channelCtl.Delete)
	// [商户后台] 财务管理-渠道管理-通道管理-列表
	post("/merchant/finance/channel/list", channelCtl.List)
	// [商户后台] 通道列表数据缓存
	get("/merchant/finance/channel/cache", channelCtl.Cache)
	// [商户后台] 财务管理-渠道管理-通道管理-启用/停用
	post("/merchant/finance/channel/updatestate", channelCtl.UpdateState)

	// [商户后台] 财务管理-渠道管理-会员等级通道-新增
	post("/merchant/finance/vip/insert", vipCtl.Insert)
	// [商户后台] 财务管理-渠道管理-会员等级通道-修改
	post("/merchant/finance/vip/update", vipCtl.Update)
	// [商户后台] 财务管理-渠道管理-会员等级通道-删除
	post("/merchant/finance/vip/delete", vipCtl.Delete)
	// [商户后台] 财务管理-渠道管理-会员等级通道-列表
	post("/merchant/finance/vip/list", vipCtl.List)
	// [商户后台] 财务管理-渠道管理-会员等级通道-启用/停用
	post("/merchant/finance/vip/updatestate", vipCtl.UpdateState)

	// [商户后台] 财务管理-渠道管理-通道银行管理-新增
	post("/merchant/finance/bank/insert", bankCtl.Insert)
	// [商户后台] 财务管理-渠道管理-通道银行管理-修改
	post("/merchant/finance/bank/update", bankCtl.Update)
	// [商户后台] 财务管理-渠道管理-通道银行管理-列表
	post("/merchant/finance/bank/list", bankCtl.List)
	// [商户后台] 财务管理-渠道管理-通道银行管理-启用/停用
	post("/merchant/finance/bank/updatestate", bankCtl.UpdateState)

	// [商户后台] 财务管理-渠道管理-会员信用等级-新增
	post("/merchant/finance/credit/insert", creditCtl.Insert)
	// [商户后台] 财务管理-渠道管理-会员信用等级-修改
	post("/merchant/finance/credit/update", creditCtl.Update)
	// [商户后台] 财务管理-渠道管理-会员信用等级-列表
	post("/merchant/finance/credit/list", creditCtl.List)
	// [商户后台] 财务管理-渠道管理-会员信用等级-启用/停用
	post("/merchant/finance/credit/updatestate", creditCtl.UpdateState)

	// [商户后台] 财务管理-渠道管理-会员信用等级-新增会员
	post("/merchant/finance/membercredit/insert", creditCtl.MemberInsert)
	// [商户后台] 财务管理-渠道管理-会员信用等级-列表会员
	post("/merchant/finance/membercredit/list", creditCtl.MemberList)
	// [商户后台] 财务管理-渠道管理-会员信用等级-删除会员
	post("/merchant/finance/membercredit/delete", creditCtl.MemberDelete)

	// [商户后台] 财务管理-渠道管理-会员锁定-新增
	post("/merchant/finance/memberlock/insert", lockCtl.MemberInsert)
	// [商户后台] 财务管理-渠道管理-会员锁定-列表
	post("/merchant/finance/memberlock/list", lockCtl.MemberList)
	// [商户后台] 财务管理-渠道管理-会员锁定-启用
	post("/merchant/finance/memberlock/updatestate", lockCtl.UpdateState)

	// [商户后台] 财务管理-渠道管理-通道类型管理-列表
	get("/merchant/finance/tunnel/list", tunnelCtl.List)
	// [商户后台] 财务管理-渠道管理-通道类型管理-修改
	post("/merchant/finance/tunnel/update", tunnelCtl.Update)
	// [商户后台] 财务管理-提款管理-会员列表-提款
	post("/merchant/finance/withdraw/memberlist", wdCtl.MemberWithdrawList)
	// [商户后台] 财务管理-提款管理-提款列表
	post("/merchant/finance/withdraw/financelist", wdCtl.FinanceReviewList)
	// [商户后台] 财务管理-提款管理-提款历史记录
	post("/merchant/finance/withdraw/historylist", wdCtl.HistoryList)
	// [商户后台] 财务管理-提款管理-拒绝
	post("/merchant/finance/withdraw/reject", wdCtl.ReviewReject)
	// [商户后台] 财务管理-提款管理-人工出款（手动代付， 手动出款）
	post("/merchant/finance/withdraw/review", wdCtl.Review)
	// [商户后台] 财务管理-提款管理-代付失败
	post("/merchant/finance/withdraw/automatic/failed", wdCtl.AutomaticFailed)

	// [商户后台] 风控管理-提款审核-待领取列表
	post("/merchant/finance/withdraw/waitreceive", wdCtl.RiskWaitConfirmList)
	// [商户后台] 风控管理-提款审核-待领取列表-银行卡交易记录统计
	post("/merchant/finance/withdraw/cardrecord", wdCtl.BankCardWithdrawRecord)
	// [商户后台] 风控管理-提款审核-待审核列表
	post("/merchant/finance/withdraw/waitreview", wdCtl.RiskReviewList)
	// [商户后台] 风控管理-提款审核-待审核列表-通过
	post("/merchant/finance/withdraw/reviewpass", wdCtl.RiskReview)
	// [商户后台] 风控管理-提款审核-待审核列表-拒绝
	post("/merchant/finance/withdraw/reviewreject", wdCtl.RiskReviewReject)
	// [商户后台] 风控管理-提款审核-待审核列表-挂起
	post("/merchant/finance/withdraw/hangup", wdCtl.HangUp)
	// [商户后台] 风控管理-提款审核-挂起列表
	post("/merchant/finance/withdraw/hanguplist", wdCtl.HangUpList)
	// [商户后台] 风控管理-提款审核-待审核列表-修改领取人
	post("/merchant/finance/withdraw/receiveupdate", wdCtl.ConfirmNameUpdate)
	// [商户后台] 风控管理-提款审核-挂起列表-领取
	post("/merchant/finance/withdraw/receive", wdCtl.ConfirmName)
	// [商户后台] 风控管理-提款审核-历史记录列表
	post("/merchant/finance/withdraw/riskhistory", wdCtl.RiskHistory)

	// [商户后台] 会员列表-存款管理-存款信息
	get("/merchant/finance/deposit/detail", depositCtl.Detail)
	// [商户后台] 财务管理-存款管理-入款订单列表/补单审核列表
	get("/merchant/finance/deposit/list", depositCtl.List)
	// [商户后台] 财务管理-存款管理-历史记录
	get("/merchant/finance/deposit/history", depositCtl.History)
	// [商户后台] 财务管理-存款管理-存款补单
	post("/merchant/finance/deposit/manual", depositCtl.Manual)
	// [商户后台] 财务管理-存款管理-补单审核
	post("/merchant/finance/deposit/review", depositCtl.Review)
	// [商户后台] 财务管理-手动下分
	post("/merchant/finance/deposit/reduce", depositCtl.Reduce)
	// [商户后台] 财务管理-存款管理-USDT存款
	post("/merchant/finance/deposit/usdt/list", depositCtl.USDTList)
	// [商户后台] 财务管理-存款管理-线下转卡-入款订单
	post("/merchant/finance/deposit/manual/list", depositCtl.Offline)
	// [商户后台] 财务管理-存款管理-线下转卡-确认金额待审核
	post("/merchant/finance/deposit/manual/reviewing", depositCtl.OfflineToReview)
	// [商户后台] 财务管理-存款管理-线下转卡-审核
	post("/merchant/finance/deposit/manual/review", depositCtl.OfflineReview)
	// [商户后台] 财务管理-存款管理-线下USDT-确认金额待审核
	post("/merchant/finance/deposit/usdt/reviewing", depositCtl.OfflineUSDT)
	// [商户后台] 财务管理-存款管理-线下USDT-审核
	post("/merchant/finance/deposit/usdt/review", depositCtl.OfflineUSDTReview)

	// [商户后台] 财务管理-渠道管理-usdt汇率设置
	post("/merchant/finance/usdt/setrate", usdtCtl.SetRate)
	// [商户后台] 财务管理-渠道管理-usdt汇率获取
	get("/merchant/finance/usdt/getrate", usdtCtl.GetRate)
	// [商户后台] 财务管理-存款管理-线下转卡-添加银行卡
	post("/merchant/finance/bankcard/add", bankCardCtl.Insert)
	// [商户后台] 财务管理-存款管理-线下转卡-列表银行卡
	post("/merchant/finance/bankcard/list", bankCardCtl.List)
	// [商户后台] 财务管理-存款管理-线下转卡-更新银行卡
	post("/merchant/finance/bankcard/update", bankCardCtl.Update)
	// [商户后台] 财务管理-存款管理-线下转卡-删除银行卡
	get("/merchant/finance/bankcard/delete", bankCardCtl.Delete)
	// [商户后台] 财务管理-存款管理-线下usdt-设置收款地址
	post("/merchant/finance/usdt/settrc", usdtCtl.SetTRC)
	// [商户后台] 财务管理-存款管理-线下usdt-获取收款地址
	get("/merchant/finance/usdt/gettrc", usdtCtl.GetTRC)

	// [商户后台] 风控管理-风控配置-接单控制-关闭自动派单
	get("/merchant/finance/risks/close", risksCtl.CloseAuto)
	// [商户后台] 风控管理-风控配置-接单控制-开启自动派单
	get("/merchant/finance/risks/open", risksCtl.OpenAuto)
	// [商户后台] 风控管理-风控配置-获取自动派单状态
	get("/merchant/finance/risks/state", risksCtl.State)
	// [商户后台] 风控管理-风控配置-获取自动派单人员的列表
	get("/merchant/finance/risks/list", risksCtl.List)
	// [商户后台] 风控管理-风控配置-设置接单数量
	get("/merchant/finance/risks/setnumer", risksCtl.SetNumber)
	// [商户后台] 风控管理-风控配置-领取人列表
	get("/merchant/finance/risks/receives", risksCtl.Receives)
	// [商户后台] 风控管理-风控配置-领取人数量
	get("/merchant/finance/risks/number", risksCtl.Number)
	// [商户后台] 风控管理-风控配置-设置同设备号注册数量
	post("/merchant/finance/risks/setregmax", risksCtl.SetRegMax)
	// [商户后台] 风控管理-风控配置-获取同设备号注册数量
	get("/merchant/finance/risks/regmax", risksCtl.RegMax)

	return router
}

// get is a shortcut for router.GET(path string, handle fasthttp.RequestHandler)
func get(path string, handle fasthttp.RequestHandler) {
	router.GET(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
}

// head is a shortcut for router.HEAD(path string, handle fasthttp.RequestHandler)
func head(path string, handle fasthttp.RequestHandler) {
	router.HEAD(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
}

// options is a shortcut for router.OPTIONS(path string, handle fasthttp.RequestHandler)
func options(path string, handle fasthttp.RequestHandler) {
	router.OPTIONS(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
}

// post is a shortcut for router.POST(path string, handle fasthttp.RequestHandler)
func post(path string, handle fasthttp.RequestHandler) {
	router.POST(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
}

// put is a shortcut for router.PUT(path string, handle fasthttp.RequestHandler)
func put(path string, handle fasthttp.RequestHandler) {
	router.PUT(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
}

// patch is a shortcut for router.PATCH(path string, handle fasthttp.RequestHandler)
func patch(path string, handle fasthttp.RequestHandler) {
	router.PATCH(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
}

// delete is a shortcut for router.DELETE(path string, handle fasthttp.RequestHandler)
func delete(path string, handle fasthttp.RequestHandler) {
	router.DELETE(path, fasthttp.TimeoutHandler(handle, ApiTimeout, ApiTimeoutMsg))
}
