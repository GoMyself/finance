package model

import (
	"crypto/sha256"
	"errors"
	"finance/contrib/helper"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

const (
	yfbMomo       = "0"
	yfbZalo       = "1"
	yfbOnline     = "3"
	yfbRemit      = "4"
	yfbUnionPay   = "2"
	yfbViettelPay = "5"
)

type yfbConf struct {
	AppID          string
	Name           string
	Domain         string
	Key            string
	Secret         string
	PayNotify      string
	PayReturn      string
	WithdrawNotify string
	Channel        map[string]string
}

type fypPayResp struct {
	Code      int    `json:"code"`
	TradeNo   string `json:"tradeNo"`
	TargetURL string `json:"targetUrl"`
}

type yfbWithdrawResp struct {
	Code    int    `json:"code"`
	TradeNo string `json:"tradeNo"`
}

//YfbPayment 优付宝
type YfbPayment struct {
	conf yfbConf
}

//Name 渠道名称
func (that *YfbPayment) Name() string {
	return that.conf.Name
}

//New 初始化配置
func (that *YfbPayment) New() {

	//appID := "9034"
	//payKey := "Tp6i_yxTh0KnxomRC3QLkg"
	//paySecret := "tLf3zaGFbUmNaVSz9T1yzg"
	//
	//if meta.IsDev {
	//	appID = "9033"
	//	payKey = "WFsGgjnjc0Ch9r6IAKR1tw"
	//	paySecret = "221IiFIzU0avMRzc8IlbeA"
	//}

	appID := meta.Finance["yfb"]["app_id"].(string)
	payKey := meta.Finance["yfb"]["key"].(string)
	paySecret := meta.Finance["yfb"]["paySecret"].(string)

	that.conf = yfbConf{
		AppID:          appID,
		Name:           "YFB",
		Domain:         "https://vn.pasvn.com",
		Key:            payKey,
		Secret:         paySecret,
		PayNotify:      "%s/finance/callback/yfbd",
		WithdrawNotify: "%s/finance/callback/yfbw",
		Channel: map[string]string{
			"momo":       yfbMomo,
			"zalo":       yfbZalo,
			"online":     yfbOnline,
			"remit":      yfbRemit,
			"unionpay":   yfbUnionPay,
			"viettelpay": yfbViettelPay,
		},
	}

}

//Pay 发起支付请求
func (that *YfbPayment) Pay(orderId, ch, amount, bid string) (paymentDepositResp, error) {
	data := paymentDepositResp{}

	cno, ok := that.conf.Channel[ch]
	if !ok {
		return data, errors.New(helper.ChannelNotExist)
	}

	now := time.Now()
	params := map[string]string{
		"merchantNo": that.conf.AppID,                                  // 商户编号
		"orderNo":    orderId,                                          // 商户订单号
		"channelNo":  cno,                                              // 纯数字格式; MomoPay:0 | ZaloPay:1 | 银行扫码:2 | 直連:3 | 网关:4 |VTPay:5
		"amount":     fmt.Sprintf("%s000", amount),                     // 订单金额
		"bankName":   bid,                                              // 银行名称 (用于银行扫码（通道2）,直連（通道3） 的收款账户分配)
		"datetime":   now.Format("2006-01-02 15:04:05"),                // 日期时间 (格式:2018-01-01 23:59:59)
		"notifyUrl":  fmt.Sprintf(that.conf.PayNotify, meta.Fcallback), // 异步通知地址
		"time":       fmt.Sprintf("%d", now.Unix()),                    // 时间戳
		"appSecret":  that.conf.Secret,                                 //
		"discount":   "",                                               //
		"extra":      "",                                               //
		"userNo":     "",                                               //
	}

	params["sign"] = that.sign(params, "deposit")

	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}

	uri := fmt.Sprintf("%s/order/create", that.conf.Domain)

	res, err := httpDoTimeout("yfb", []byte(formData.Encode()), "POST", uri, nil, time.Second*8)
	if err != nil {
		fmt.Println("yfb uri = ", uri)
		fmt.Println("yfb httpDoTimeout err = ", err)

		return data, errors.New(helper.PayServerErr)
	}

	var rp fypPayResp
	if err := helper.JsonUnmarshal(res, &rp); err != nil {
		return data, fmt.Errorf("json format err: %s", err.Error())
	}

	if rp.Code != 0 {
		return data, fmt.Errorf("an 3rd-party error occurred")
	}

	data.Addr = rp.TargetURL
	data.OrderID = rp.TradeNo

	return data, nil
}

//Withdraw 发起提现
func (that *YfbPayment) Withdraw(arg WithdrawAutoParam) (paymentWithdrawalRsp, error) {

	data := paymentWithdrawalRsp{}

	dateTime := arg.Ts.Format("2006-01-02 15:04:05")
	t := fmt.Sprintf("%d", arg.Ts.Unix())

	params := map[string]string{
		"merchantNo":  that.conf.AppID,                                       // 商户编号
		"orderNo":     arg.OrderID,                                           // 商户订单号
		"amount":      arg.Amount,                                            // 订单金额
		"name":        arg.CardName,                                          // 收款人姓名
		"bankName":    arg.BankCode,                                          // 收款银行名称
		"bankAccount": arg.CardNumber,                                        // 收款银行账号
		"bankBranch":  "",                                                    // 收款银行支行 (可选；提供此项可加速入账。示例：NGUYEN)
		"datetime":    dateTime,                                              // 日期时间 (格式:2018-01-01 23:59:59)
		"notifyUrl":   fmt.Sprintf(that.conf.WithdrawNotify, meta.Fcallback), // 异步通知地址
		"time":        t,                                                     // 时间戳
		"appSecret":   that.conf.Secret,                                      //
		"memo":        "",                                                    // 收款附言 (可选)
		"mobile":      "",                                                    // 收款通知手机号 (可选；如果收款银行支持，则会发送手机短信转账通知。)
		"reverseUrl":  "",                                                    // 冲正回调地址 (可选；当代付触发银行冲正时，平台将向此URL地址发送异步通知。建议使用 https。不提供此参数，则冲正由客服人工处理)
		"extra":       "",                                                    // 附加信息 (可选；回调时原样返回)
	}

	params["sign"] = that.sign(params, "withdraw")

	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}

	uri := fmt.Sprintf("%s/payout/create", that.conf.Domain)

	v, err := httpDoTimeout("yfb", []byte(formData.Encode()), "POST", uri, nil, time.Second*8)
	if err != nil {
		return data, err
	}

	var res yfbWithdrawResp
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

//PayCallBack 支付回调
func (that *YfbPayment) PayCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	status := string(fctx.PostArgs().Peek("status"))   // PAID(已付); MANUAL PAID (已补单)
	orderNo := string(fctx.PostArgs().Peek("orderNo")) // 商户单号
	amount := string(fctx.PostArgs().Peek("amount"))   // 订单金额
	sign := string(fctx.PostArgs().Peek("sign"))       // 签名

	data := paymentCallbackResp{
		State: DepositConfirming,
		Sign:  sign,
	}

	switch status {
	case "PAID", "MANUAL PAID":
		data.State = DepositSuccess
	default:
		return data, fmt.Errorf("unknown status: [%s]", status)
	}

	// check signature
	args := map[string]string{
		"status":    status,
		"tradeNo":   string(fctx.PostArgs().Peek("tradeNo")),
		"orderNo":   orderNo,
		"userNo":    string(fctx.PostArgs().Peek("userNo")),
		"userName":  string(fctx.PostArgs().Peek("userName")),
		"channelNo": string(fctx.PostArgs().Peek("channelNo")),
		"amount":    amount,
		"discount":  string(fctx.PostArgs().Peek("discount")),
		"lucky":     string(fctx.PostArgs().Peek("lucky")),
		"paid":      string(fctx.PostArgs().Peek("paid")),
		"extra":     string(fctx.PostArgs().Peek("extra")),
	}

	if that.sign(args, "deposit") != data.Sign {
		return data, fmt.Errorf("invalid sign")
	}

	data.OrderID = orderNo
	data.Amount = amount

	return data, nil
}

//WithdrawCallBack 提款回调
func (that *YfbPayment) WithdrawCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	status := string(fctx.PostArgs().Peek("status"))   // PAID(已付); MANUAL PAID (已补单)
	orderNo := string(fctx.PostArgs().Peek("orderNo")) // 商户单号
	amount := string(fctx.PostArgs().Peek("amount"))   // 订单金额
	sign := string(fctx.PostArgs().Peek("sign"))       // 签名

	data := paymentCallbackResp{
		State: WithdrawDealing,
		Sign:  sign,
	}

	switch status {
	case "PAID", "MANUAL PAID":
		data.State = WithdrawSuccess
	case "CANCELLED":
		data.State = WithdrawAutoPayFailed
	default:
		return data, fmt.Errorf("unknown status: [%s]", status)
	}

	// check signature
	args := map[string]string{
		"status":      status,
		"tradeNo":     string(fctx.PostArgs().Peek("tradeNo")),
		"orderNo":     orderNo,
		"amount":      amount,
		"name":        string(fctx.PostArgs().Peek("name")),
		"bankName":    string(fctx.PostArgs().Peek("bankName")),
		"bankAccount": string(fctx.PostArgs().Peek("bankAccount")),
		"bankBranch":  string(fctx.PostArgs().Peek("bankBranch")),
		"memo":        string(fctx.PostArgs().Peek("memo")),
		"mobile":      string(fctx.PostArgs().Peek("mobile")),
		"fee":         string(fctx.PostArgs().Peek("fee")),
		"extra":       string(fctx.PostArgs().Peek("extra")),
	}

	if that.sign(args, "withdraw") != data.Sign {
		return data, fmt.Errorf("invalid sign")
	}

	data.OrderID = orderNo
	data.Amount = amount

	return data, nil
}

func (that *YfbPayment) sign(args map[string]string, method string) string {

	qs := ""
	keys := make([]string, 0)

	for k := range args {
		if method == "deposit" {
			switch k {
			case "userName", "channelNo", "amountBeforeFixed", "payeeName", "appSecret", "bankName":
				continue
			}
		}

		if method == "withdraw" {
			switch k {
			case "bankBranch", "memo", "appSecret":
				continue
			}
		}

		keys = append(keys, k)
	}

	sort.Strings(keys)
	for _, v := range keys {
		qs += fmt.Sprintf("%s=%s&", v, args[v])
	}
	qs = qs[:len(qs)-1] + that.conf.Key

	s256 := fmt.Sprintf("%x", sha256.Sum256([]byte(qs)))

	return strings.ToUpper(helper.GetMD5Hash(s256))
}
