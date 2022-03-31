package model

import (
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
	vtMomo       = "VT_MOMO_QR"
	vtViettelPay = "VT_VIETTEL_QR"
	vtZaloPay    = "VT_ZALO_QR"
	vtOnline     = "VT_ONLINE_BANK"
	vtBankQr     = "VT_BANK_QR"
)

type vtPayConf struct {
	MerchantNo     string
	Key            string
	Name           string
	Domain         string
	PayNotify      string
	WithdrawNotify string
	Channel        map[paymentChannel]string
}

type VtPayment struct {
	Conf vtPayConf
}

type vtPayResp struct {
	Code      int    `json:"code"`
	TradeNo   string `json:"tradeNo"`
	TargetURL string `json:"targetUrl"`
}

type vtPayWithdrawResp struct {
	Code    int    `json:"code"`
	TradeNo string `json:"tradeNo"`
}

func (that *VtPayment) New() {

	appID := meta.Finance["vt"]["app_id"].(string)
	payKey := meta.Finance["vt"]["key"].(string)
	apiUrl := meta.Finance["vt"]["api"].(string)
	that.Conf = vtPayConf{
		MerchantNo:     appID,
		Key:            payKey,
		Name:           "VtPay",
		Domain:         apiUrl,
		PayNotify:      "%s/finance/callback/wd",
		WithdrawNotify: "%s/finance/callback/ww",
		Channel: map[paymentChannel]string{
			momo:       vtMomo,
			unionpay:   vtBankQr,
			online:     vtOnline,
			zalo:       vtZaloPay,
			viettelpay: vtViettelPay,
		},
	}
}

func (that *VtPayment) Name() string {
	return that.Conf.Name
}

func (that *VtPayment) Pay(log *paymentTDLog, ch paymentChannel, amount, bid string) (paymentDepositResp, error) {

	data := paymentDepositResp{}

	cno, ok := that.Conf.Channel[ch]
	if !ok {
		return data, errors.New(helper.ChannelNotExist)
	}

	recs := map[string]string{
		"merchantNo": that.Conf.MerchantNo,                             // 商户编号
		"orderNo":    log.OrderID,                                      // 商户订单号
		"channel":    cno,                                              // 纯数字格式; MomoPay:0 | ZaloPay:1 | 银行扫码:2 | 直連:3 | 网关:4 |VTPay:5
		"amount":     fmt.Sprintf("%s000", amount),                     // 订单金额
		"bankCode":   bid,                                              // 银行编号VCB
		"notifyUrl":  fmt.Sprintf(that.Conf.PayNotify, meta.Fcallback), // 异步通知地址 		//
	}

	recs["sign"] = that.sign(recs, "deposit")

	formData := url.Values{}
	for k, v := range recs {
		formData.Set(k, v)
	}
	header := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	uri := fmt.Sprintf("%s/vtpay/request", that.Conf.Domain)
	v, err := httpDoTimeout([]byte(formData.Encode()), "POST", uri, header, time.Second*8, log)
	if err != nil {
		return data, err
	}

	data.Addr = string(v)
	data.OrderID = log.OrderID

	return data, nil
}

func (that *VtPayment) Withdraw(log *paymentTDLog, arg WithdrawAutoParam) (paymentWithdrawalRsp, error) {

	data := paymentWithdrawalRsp{}
	params := map[string]string{
		"merchantNo":    that.Conf.MerchantNo,                                  // 商户编号
		"orderNo":       arg.OrderID,                                           // 商户订单号
		"amount":        arg.Amount,                                            // 订单金额
		"accountName":   arg.CardName,                                          // 收款人姓名
		"bankId":        arg.BankCode,                                          // 收款银行名称
		"accountNumber": arg.CardNumber,                                        // 收款银行账号
		"notifyUrl":     fmt.Sprintf(that.Conf.WithdrawNotify, meta.Fcallback), // 异步通知地址
	}

	params["sign"] = that.sign(params, "withdraw")
	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}
	uri := fmt.Sprintf("%s/vtpay/cashout/transfer", that.Conf.Domain)

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

func (that *VtPayment) PayCallBack(ctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	params := make(map[string]string)
	ctx.PostArgs().VisitAll(func(key, value []byte) {
		params[string(key)] = string(value)
	})

	data := paymentCallbackResp{
		State: DepositConfirming,
		Sign:  string(ctx.PostArgs().Peek("sign")),
	}

	if !valid(params, []string{"merchantNo", "orderNo", "referenceNo", "amount", "channel", "successAmount", "sign"}) {
		return data, fmt.Errorf("param err: [%v]", params)
	}

	data.State = DepositSuccess

	if that.sign(params, "depositCall") != params["sign"] {
		return data, fmt.Errorf("invalid sign")
	}

	data.OrderID = params["orderNo"]
	data.Amount = params["successAmount"]

	return data, nil
}

func (that *VtPayment) WithdrawCallBack(ctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	params := make(map[string]string)
	ctx.PostArgs().VisitAll(func(key, value []byte) {
		params[string(key)] = string(value)
	})

	data := paymentCallbackResp{
		State: WithdrawDealing,
		Sign:  string(ctx.PostArgs().Peek("sign")),
	}

	if !valid(params, []string{"merchantNo", "status", "amount", "orderNo", "referenceNo", "sign"}) {
		return data, fmt.Errorf("param err: [%v]", params)
	}

	switch params["status"] {
	case "1":
		data.State = WithdrawSuccess
	case "2", "3":
		data.State = WithdrawAutoPayFailed
	default:
		return data, fmt.Errorf("unknown status: [%s]", params["status"])
	}

	if that.sign(params, "withdrawCall") != data.Sign {
		return data, fmt.Errorf("invalid sign")
	}

	data.OrderID = params["orderNo"]
	data.Amount = params["amount"]

	return data, nil
}

func (that *VtPayment) sign(args map[string]string, method string) string {

	qs := ""
	keys := make([]string, 0)

	for k := range args {
		if method == "deposit" {
			switch k {
			case "merchantNo", "orderNo", "amount", "channel":
				continue
			}
		}

		if method == "depositCall" {
			switch k {
			case "merchantNo", "orderNo", "referenceNo", "amount", "channel":
				continue
			}
		}

		if method == "withdraw" {
			switch k {
			case "merchantNo", "orderNo", "amount", "accountNumber":
				continue
			}
		}

		if method == "withdrawCall" {
			switch k {
			case "merchantNo", "orderNo":
				continue
			}
		}

		keys = append(keys, k)
	}

	sort.Strings(keys)
	for _, v := range keys {
		qs += fmt.Sprintf("%s=%s&", v, args[v])
	}
	qs = qs + "key=" + that.Conf.Key

	return strings.ToLower(helper.GetMD5Hash(qs))
}
