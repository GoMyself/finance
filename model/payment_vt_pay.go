package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

const (
	vtMomo       = "VT_MOMO_QR"
	vtViettelPay = "VT_VIETTEL_QR"
	vtZaloPay    = "VT_ZALO_QR"
	vtOnline     = "VT_ONLINE_BANK"
	vtBankQr     = "VT_BANK_QR"
	vtCard       = "VT_CARD_RECHARGE"
)

type vtPayConf struct {
	MerchantNo     string
	Key            string
	Name           string
	Domain         string
	PayNotify      string
	WithdrawNotify string
	Channel        map[string]string
}

type VtPayment struct {
	Conf vtPayConf
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
		PayNotify:      "%s/finance/callback/vtd",
		WithdrawNotify: "%s/finance/callback/vtw",
		Channel: map[string]string{
			"1":  vtMomo,
			"8":  vtBankQr,
			"3":  vtOnline,
			"2":  vtZaloPay,
			"6":  vtViettelPay,
			"15": vtCard,
		},
	}
}

func (that *VtPayment) Name() string {
	return that.Conf.Name
}

func (that *VtPayment) Pay(orderId, ch, amount, bid string) (paymentDepositResp, error) {

	data := paymentDepositResp{}
	cno, ok := that.Conf.Channel[ch]
	if !ok {
		return data, errors.New(helper.ChannelNotExist)
	}

	recs := map[string]string{
		"merchantNo": that.Conf.MerchantNo,                             // 商户编号
		"orderNo":    orderId,                                          // 商户订单号
		"channel":    cno,                                              // 纯数字格式; MomoPay:0 | ZaloPay:1 | 银行扫码:2 | 直連:3 | 网关:4 |VTPay:5
		"amount":     fmt.Sprintf("%s000", amount),                     // 订单金额
		"bankCode":   bid,                                              // 银行编号VCB
		"notifyUrl":  fmt.Sprintf(that.Conf.PayNotify, meta.Fcallback), // 异步通知地址 		//
	}

	recs["sign"] = that.sign(recs, "deposit")

	uri := fmt.Sprintf("%s/vtpay/request?%s", that.Conf.Domain, paramEncode(recs))
	fmt.Println(uri)
	data.Addr = uri
	data.OrderID = orderId

	return data, nil
}

func (that *VtPayment) Withdraw(arg WithdrawAutoParam) (paymentWithdrawalRsp, error) {

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
	v, err := httpDoTimeout("vt pay", []byte(formData.Encode()), "POST", uri, headers, time.Second*8)
	if err != nil {
		fmt.Println("vt uri = ", uri)
		fmt.Println("vt httpDoTimeout err = ", err)
		fmt.Println("vnpay body = ", string(v))
		return data, errors.New(helper.PayServerErr)
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

func (that *VtPayment) PayCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	params := make(map[string]string)
	fctx.PostArgs().VisitAll(func(key, value []byte) {
		params[string(key)] = string(value)
	})

	data := paymentCallbackResp{
		State: DepositConfirming,
		Sign:  string(fctx.PostArgs().Peek("sign")),
	}

	if !valid(params, []string{"merchantNo", "orderNo", "referenceNo", "amount", "channel", "successAmount", "sign"}) {
		return data, fmt.Errorf("param err: [%v]", params)
	}

	data.State = DepositSuccess

	if that.sign(params, "depositCall") != params["sign"] {
		return data, fmt.Errorf("invalid sign")
	}

	data.OrderID = params["orderNo"]
	data.Amount = params["amount"]

	return data, nil
}

func (that *VtPayment) WithdrawCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	params := make(map[string]string)
	fctx.PostArgs().VisitAll(func(key, value []byte) {
		params[string(key)] = string(value)
	})

	data := paymentCallbackResp{
		State: WithdrawDealing,
		Sign:  string(fctx.PostArgs().Peek("sign")),
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

	if method == "deposit" {

		qs = fmt.Sprintf(`merchantNo=%s&orderNo=%s&amount=%s&channel=%s&key=%s`, args["merchantNo"],
			args["orderNo"], args["amount"], args["channel"], that.Conf.Key)
	}

	if method == "depositCall" {
		qs = fmt.Sprintf(`merchantNo=%s&orderNo=%s&referenceNo=%s&amount=%s&channel=%s&key=%s`, args["merchantNo"],
			args["orderNo"], args["referenceNo"], args["amount"], args["channel"], that.Conf.Key)
	}

	if method == "withdraw" {
		qs = fmt.Sprintf(`merchantNo=%s&orderNo=%s&amount=%s&accountNumber=%s&key=%s`, args["merchantNo"],
			args["orderNo"], args["amount"], args["accountNumber"], that.Conf.Key)
	}

	if method == "withdrawCall" {
		qs = fmt.Sprintf(`merchantNo=%s&orderNo=%s&referenceNo=%s&amount=%s&key=%s`, args["merchantNo"],
			args["orderNo"], args["referenceNo"], args["amount"], that.Conf.Key)
	}

	fmt.Printf("sign content:" + qs)
	return strings.ToLower(helper.GetMD5Hash(qs))
}
