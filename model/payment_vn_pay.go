package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"
	"github.com/valyala/fasthttp"
	"strings"
	"time"
)

const (
	vnOnline = "Y"
)

type vnPayConf struct {
	AppID          string
	Merchan        string
	MerchanNo      string
	PayKey         string
	PaySecret      string
	Name           string
	Domain         string
	Key            string
	PayNotify      string
	WithdrawNotify string
	Channel        map[paymentChannel]string
}

type VnPayment struct {
	Conf vnPayConf
}

type vnPayResp struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		OrderNo string `json:"orderNo"`
		Link    string `json:"link"`
	} `json:"data"`
}

type vnPayWithdrawResp struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data string `json:"data"`
}

func (that *VnPayment) New() {

	appID := meta.Finance["vn"]["app_id"].(string)
	merchan := meta.Finance["vn"]["merchan"].(string)
	merchanNo := meta.Finance["vn"]["merchan_no"].(string)
	payKey := meta.Finance["vn"]["key"].(string)
	apiUrl := meta.Finance["vn"]["api"].(string)
	that.Conf = vnPayConf{
		AppID:          appID,
		Merchan:        merchan,
		MerchanNo:      merchanNo,
		PayKey:         payKey,
		Name:           "VnPay",
		Domain:         apiUrl,
		PayNotify:      "%s/finance/callback/vnd",
		WithdrawNotify: "%s/finance/callback/vnw",
		Channel: map[paymentChannel]string{
			online: vnOnline,
		},
	}
}

func (that *VnPayment) Name() string {
	return that.Conf.Name
}

func (that *VnPayment) Pay(log *paymentTDLog, ch paymentChannel, amount, bid string) (paymentDepositResp, error) {

	data := paymentDepositResp{}

	cno, ok := that.Conf.Channel[ch]
	if !ok {
		return data, errors.New(helper.ChannelNotExist)
	}

	now := time.Now()
	recs := map[string]string{
		"merchantNo":  that.Conf.MerchanNo,                              // 商户编号
		"channelCode": bid,                                              // 银行名称 (用于银行扫码（通道2）,直連（通道3） 的收款账户分配)
		"orderNo":     log.OrderID,                                      // 商户订单号
		"bankDirct":   cno,                                              // 纯数字格式; MomoPay:0 | ZaloPay:1 | 银行扫码:2 | 直連:3 | 网关:4 |VTPay:5
		"currency":    "VND",                                            //
		"amount":      fmt.Sprintf("%s000.00", amount),                  // 订单金额
		"notifyUrl":   fmt.Sprintf(that.Conf.PayNotify, meta.Fcallback), // 异步通知地址
	}
	tp := fmt.Sprintf("%d", now.UnixMilli())
	fmt.Println(tp)
	recs["timestamp"] = tp
	recs["sign"] = that.sign(recs, "deposit")
	delete(recs, "timestamp")
	body, err := helper.JsonMarshal(recs)
	if err != nil {
		return data, errors.New(helper.FormatErr)
	}

	header := map[string]string{
		"Content-Type": "application/json",
		"Nonce":        helper.MD5Hash(helper.GenId()),
		"Timestamp":    tp,
		"x-Request-Id": helper.GenId(),
	}

	uri := fmt.Sprintf("%s/v1/api/online/ebank/%s/%s/%s", that.Conf.Domain, that.Conf.AppID, that.Conf.Merchan, log.OrderID)
	v, err := httpDoTimeout(body, "POST", uri, header, time.Second*8, log)
	if err != nil {
		return data, err
	}

	var res vnPayResp

	if err = helper.JsonUnmarshal(v, &res); err != nil {
		return data, fmt.Errorf("json format err: %s", err.Error())
	}

	if res.Code != "0000" {
		return data, fmt.Errorf("an 3rd-party error occurred")
	}

	data.Addr = res.Data.Link
	data.OrderID = res.Data.OrderNo

	return data, nil
}

func (that *VnPayment) Withdraw(log *paymentTDLog, arg WithdrawAutoParam) (paymentWithdrawalRsp, error) {

	data := paymentWithdrawalRsp{}
	params := map[string]string{
		"merchantNo":    that.Conf.MerchanNo, // 商户编号
		"channelCode":   arg.BankCode,        // 收款银行名称
		"orderNo":       arg.OrderID,         // 商户订单号
		"currency":      "VND",
		"amount":        fmt.Sprintf("%s.00", arg.Amount),                      // 订单金额
		"payee":         arg.CardName,                                          // 收款人姓名
		"payeeBankCard": arg.CardNumber,                                        // 收款银行账号
		"notifyUrl":     fmt.Sprintf(that.Conf.WithdrawNotify, meta.Fcallback), // 异步通知地址
		"verifyUrl":     "",                                                    // 验证订单地址,若提供则,我方 post 请 求验证,默认返回 {“code”:”0000”}
	}

	params["sign"] = that.sign(params, "withdraw")
	body, err := helper.JsonMarshal(params)
	if err != nil {
		return data, errors.New(helper.FormatErr)
	}
	uri := fmt.Sprintf("%s/v1/api/online/ebank/%s/%s/%s", that.Conf.Domain, that.Conf.AppID, that.Conf.Merchan, arg.OrderID)
	now := time.Now()
	tp := fmt.Sprintf("%d", now.UnixMilli())
	fmt.Println(tp)
	header := map[string]string{
		"Content-Type": "application/json",
		"Nonce":        helper.MD5Hash(helper.GenId()),
		"Timestamp":    tp,
		"x-Request-Id": helper.GenId(),
	}
	v, err := httpDoTimeout(body, "POST", uri, header, time.Second*8, log)
	if err != nil {
		return data, err
	}

	var res vnPayWithdrawResp
	err = helper.JsonUnmarshal(v, &res)
	if err != nil {
		return data, err
	}

	if res.Code != "0000" {
		return data, fmt.Errorf("an 3rd-party error occurred")
	}

	data.OrderID = res.Data

	return data, nil
}

func (that *VnPayment) PayCallBack(ctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	params := make(map[string]string)
	ctx.PostArgs().VisitAll(func(key, value []byte) {
		params[string(key)] = string(value)
	})

	data := paymentCallbackResp{
		State: DepositConfirming,
		Sign:  string(ctx.PostArgs().Peek("sign")),
	}

	if !valid(params, []string{"merchantNo", "merchantOrderNo", "channelCode", "orderNo", "currency", "amount", "status"}) {
		return data, fmt.Errorf("param err: [%v]", params)
	}

	switch params["status"] {
	case "Success":
		data.State = DepositSuccess
	default:
		return data, fmt.Errorf("unknown status: [%s]", params["status"])
	}

	if that.sign(params, "call") != params["sign"] {
		return data, fmt.Errorf("invalid sign")
	}

	data.OrderID = params["merchantOrderNo"]
	data.Amount = params["amount"]
	data.Resp = `{"code" : "0000"}`
	return data, nil
}

func (that *VnPayment) WithdrawCallBack(ctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	params := make(map[string]string)
	ctx.PostArgs().VisitAll(func(key, value []byte) {
		params[string(key)] = string(value)
	})

	data := paymentCallbackResp{
		State: WithdrawDealing,
		Sign:  string(ctx.PostArgs().Peek("sign")),
	}

	if !valid(params, []string{"merchantNo", "merchantOrderNo", "channelCode", "orderNo", "currency", "amount", "status"}) {
		return data, fmt.Errorf("param err: [%v]", params)
	}

	switch params["status"] {
	case "Success":
		data.State = WithdrawSuccess
	case "Failure":
		data.State = WithdrawAutoPayFailed
	default:
		return data, fmt.Errorf("unknown status: [%s]", params["status"])
	}

	if that.sign(params, "call") != data.Sign {
		return data, fmt.Errorf("invalid sign")
	}

	data.OrderID = params["merchantOrderNo"]
	data.Amount = params["amount"]
	data.Resp = `{"code" : "0000"}`
	return data, nil
}

func (that *VnPayment) sign(args map[string]string, method string) string {

	qs := ""

	if method == "deposit" {
		qs += fmt.Sprintf(`merchantNo=%s&channelCode=%s&orderNo=%s&currency=%s&amount=%s&notifyUrl=%s&timestamp=%s`,
			args["merchantNo"], args["channelCode"], args["orderNo"], args["currency"], args["amount"], args["notifyUrl"],
			args["timestamp"])
	}

	if method == "call" {
		qs += fmt.Sprintf(`merchantNo=%s&merchantOrderNo=%s&orderNo=%s&channelCode=%s&currency=%s&amount=%s&status=%s`,
			args["merchantNo"], args["merchantOrderNo"], args["orderNo"], args["channelCode"], args["currency"], args["amount"],
			args["status"])
	}

	if method == "withdraw" {
		qs += fmt.Sprintf(`merchantNo=%s&channelCode=%s&orderNo=%s&currency=%s&amount=%s&payee=%s&payeeBankCard=%s&notifyUrl=%s`,
			args["merchantNo"], args["channelCode"], args["orderNo"], args["currency"], args["amount"], args["payee"],
			args["payeeBankCard"], args["notifyUrl"])
	}

	qs = qs + "&appsecret=" + that.Conf.PayKey

	return strings.ToLower(helper.GetMD5Hash(helper.GetMD5Hash(helper.GetMD5Hash(qs))))
}
