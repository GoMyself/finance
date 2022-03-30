package model

import (
	"crypto/sha256"
	"errors"
	"finance/contrib/helper"
	"fmt"
	"github.com/valyala/fasthttp"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	wMomo     = "0"
	wUnionPay = "2"
	wOnline   = "3"
	wRemit    = "4"
)

type wPayConf struct {
	AppID          string
	PayKey         string
	PaySecret      string
	Name           string
	Domain         string
	Key            string
	PayNotify      string
	WithdrawNotify string
	Channel        map[paymentChannel]string
}

type WPayment struct {
	Conf wPayConf
}

type wPayResp struct {
	Code      int    `json:"code"`
	TradeNo   string `json:"tradeNo"`
	TargetURL string `json:"targetUrl"`
}

type wPayWithdrawResp struct {
	Code    int    `json:"code"`
	TradeNo string `json:"tradeNo"`
}

func (that *WPayment) New() {

	appID := meta.Finance["w"]["app_id"].(string)
	payKey := meta.Finance["w"]["key"].(string)
	paySecret := meta.Finance["w"]["paySecret"].(string)
	apiUrl := meta.Finance["w"]["api"].(string)
	that.Conf = wPayConf{
		AppID:          appID,
		PayKey:         payKey,
		PaySecret:      paySecret,
		Name:           "WPay",
		Domain:         apiUrl,
		PayNotify:      "%s/finance/callback/wd",
		WithdrawNotify: "%s/finance/callback/ww",
		Channel: map[paymentChannel]string{
			momo:     wMomo,
			unionpay: wUnionPay,
			online:   wOnline,
			remit:    wRemit,
		},
	}
}

func (that *WPayment) Name() string {
	return that.Conf.Name
}

func (that *WPayment) Pay(log *paymentTDLog, ch paymentChannel, amount, bid string) (paymentDepositResp, error) {

	data := paymentDepositResp{}

	cno, ok := that.Conf.Channel[ch]
	if !ok {
		return data, errors.New(helper.ChannelNotExist)
	}

	now := time.Now()
	recs := map[string]string{
		"merchantNo": that.Conf.AppID,                                  // 商户编号
		"orderNo":    log.OrderID,                                      // 商户订单号
		"channelNo":  cno,                                              // 纯数字格式; MomoPay:0 | ZaloPay:1 | 银行扫码:2 | 直連:3 | 网关:4 |VTPay:5
		"amount":     fmt.Sprintf("%s000", amount),                     // 订单金额
		"bankName":   bid,                                              // 银行名称 (用于银行扫码（通道2）,直連（通道3） 的收款账户分配)
		"datetime":   now.Format("2006-01-02 15:04:05"),                // 日期时间 (格式:2018-01-01 23:59:59)
		"notifyUrl":  fmt.Sprintf(that.Conf.PayNotify, meta.Fcallback), // 异步通知地址
		"time":       fmt.Sprintf("%d", now.Unix()),                    // 时间戳
		"appSecret":  that.Conf.PaySecret,                              //
		"discount":   "",                                               //
		"extra":      "",                                               //
		"userNo":     "",                                               //
	}

	recs["sign"] = that.sign(recs, "deposit")

	formData := url.Values{}
	for k, v := range recs {
		formData.Set(k, v)
	}
	header := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	uri := fmt.Sprintf("%s/order/create", that.Conf.Domain)
	v, err := httpDoTimeout([]byte(formData.Encode()), "POST", uri, header, time.Second*8, log)
	if err != nil {
		return data, err
	}

	var res wPayResp

	if err = helper.JsonUnmarshal(v, &res); err != nil {
		return data, fmt.Errorf("json format err: %s", err.Error())
	}

	if res.Code != 0 {
		return data, fmt.Errorf("an 3rd-party error occurred")
	}

	data.Addr = res.TargetURL
	data.OrderID = res.TradeNo

	return data, nil
}

func (that *WPayment) Withdraw(log *paymentTDLog, arg WithdrawAutoParam) (paymentWithdrawalRsp, error) {

	data := paymentWithdrawalRsp{}
	params := map[string]string{
		"merchantNo":  that.Conf.AppID,                                       // 商户编号
		"orderNo":     arg.OrderID,                                           // 商户订单号
		"amount":      arg.Amount,                                            // 订单金额
		"name":        arg.CardName,                                          // 收款人姓名
		"bankName":    arg.BankCode,                                          // 收款银行名称
		"bankAccount": arg.CardNumber,                                        // 收款银行账号
		"bankBranch":  "",                                                    // 收款银行支行 (可选；提供此项可加速入账。示例：NGUYEN)
		"datetime":    arg.Ts.Format("2006-01-02 15:04:05"),                  // 日期时间 (格式:2018-01-01 23:59:59)
		"notifyUrl":   fmt.Sprintf(that.Conf.WithdrawNotify, meta.Fcallback), // 异步通知地址
		"time":        fmt.Sprintf("%d", arg.Ts.Unix()),                      // 时间戳
		"appSecret":   that.Conf.PaySecret,                                   //
		"memo":        "",                                                    // 收款附言 (可选)
		"mobile":      "",                                                    // 收款通知手机号 (可选；如果收款银行支持，则会发送手机短信转账通知。)
		"reverseUrl":  fmt.Sprintf(that.Conf.WithdrawNotify, meta.Fcallback), // 异步通知地址 失败地址 冲正回调地址 (可选；当代付触发银行冲正时，平台将向此URL地址发送异步通知。建议使用 https。不提供此参数，则冲正由客服人工处理)
		"extra":       "",                                                    // 附加信息 (可选；回调时原样返回)
	}

	params["sign"] = that.sign(params, "withdraw")
	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}
	uri := fmt.Sprintf("%s/payout/create", that.Conf.Domain)

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	v, err := httpDoTimeout([]byte(formData.Encode()), "POST", uri, headers, time.Second*8, log)
	if err != nil {
		return data, err
	}

	var res wPayWithdrawResp
	err = helper.JsonUnmarshal(v, &res)
	if err != nil {
		return data, err
	}

	if res.Code != 0 {
		return data, fmt.Errorf("an 3rd-party error occurred")
	}

	data.OrderID = res.TradeNo

	return data, nil
}

func (that *WPayment) PayCallBack(ctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	params := make(map[string]string)
	ctx.PostArgs().VisitAll(func(key, value []byte) {
		params[string(key)] = string(value)
	})

	data := paymentCallbackResp{
		State: DepositConfirming,
		Sign:  string(ctx.PostArgs().Peek("sign")),
	}

	if !valid(params, []string{"sign", "status", "amount", "orderNo"}) {
		return data, fmt.Errorf("param err: [%v]", params)
	}

	switch params["status"] {
	case "PAID", "MANUAL PAID":
		data.State = DepositSuccess
	default:
		return data, fmt.Errorf("unknown status: [%s]", params["status"])
	}

	if that.sign(params, "deposit") != params["sign"] {
		return data, fmt.Errorf("invalid sign")
	}

	data.OrderID = params["orderNo"]
	data.Amount = params["amount"]

	return data, nil
}

func (that *WPayment) WithdrawCallBack(ctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	params := make(map[string]string)
	ctx.PostArgs().VisitAll(func(key, value []byte) {
		params[string(key)] = string(value)
	})

	data := paymentCallbackResp{
		State: WithdrawDealing,
		Sign:  string(ctx.PostArgs().Peek("sign")),
	}

	if !valid(params, []string{"sign", "status", "amount", "orderNo"}) {
		return data, fmt.Errorf("param err: [%v]", params)
	}

	switch params["status"] {
	case "PAID", "MANUAL PAID":
		data.State = WithdrawSuccess
	case "CANCELLED":
		data.State = WithdrawAutoPayFailed
	default:
		return data, fmt.Errorf("unknown status: [%s]", params["status"])
	}

	if that.sign(params, "withdraw") != data.Sign {
		return data, fmt.Errorf("invalid sign")
	}

	data.OrderID = params["orderNo"]
	data.Amount = params["amount"]

	return data, nil
}

func (that *WPayment) sign(args map[string]string, method string) string {

	qs := ""
	keys := make([]string, 0)

	for k := range args {
		if method == "deposit" {
			switch k {
			case "userName", "sign", "channelNo", "amountBeforeFixed", "payeeName", "appSecret", "bankName":
				continue
			}
		}

		if method == "withdraw" {
			switch k {
			case "bankBranch", "memo", "appSecret", "sign":
				continue
			}
		}

		keys = append(keys, k)
	}

	sort.Strings(keys)
	for _, v := range keys {
		qs += fmt.Sprintf("%s=%s&", v, args[v])
	}
	qs = qs[:len(qs)-1] + that.Conf.PayKey

	s256 := fmt.Sprintf("%x", sha256.Sum256([]byte(qs)))

	return strings.ToUpper(helper.GetMD5Hash(s256))
}
