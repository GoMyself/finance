package model

import (
	"github.com/valyala/fasthttp"
)

// Payment 接口
type Payment interface {
	// Name 支付通道名称
	Name() string
	// New 初始化 通道配置
	New()
	// Pay 发起支付
	Pay(log *paymentTDLog, ch paymentChannel, amount, bid string) (paymentDepositResp, error)
	// Withdraw 发起代付
	Withdraw(log *paymentTDLog, param WithdrawAutoParam) (paymentWithdrawalRsp, error)
	// PayCallBack 支付回调
	PayCallBack(ctx *fasthttp.RequestCtx) (paymentCallbackResp, error)
	// WithdrawCallBack 代付回调
	WithdrawCallBack(*fasthttp.RequestCtx) (paymentCallbackResp, error)
}
