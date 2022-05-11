package model

import (
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
	dbMomo   = "923"
	dbRemit  = "908"
	dbOnline = "907"
	//dbZalo     = "921"
	dbViettelpay = "925"
)

type dbPayConf struct {
	AppID          string
	PayKey         string
	PaySecret      string
	Name           string
	Domain         string
	Key            string
	PayNotify      string
	WithdrawNotify string
	Channel        map[string]string
}

type DbPayment struct {
	Conf dbPayConf
}

type dbPayResp struct {
	Status int `json:"status"`
	Result struct {
		Transactionid string `json:"transactionid"`
		Payurl        string `json:"payurl"`
		Points        string `json:"points"`
	} `json:"result"`
	Sign string `json:"sign"`
}

type dbPayWithdrawResp struct {
	Transactionid string `json:"transactionid"`
	Orderid       string `json:"orderid"`
	Amount        string `json:"amount"`
	RealAmount    string `json:"real_Amount"`
	Custom        string `json:"custom"`
}

type dbPayCallResp struct {
	transactionid string `json:"transactionid"`
	orderid       string `json:"orderid"`
	amount        string `json:"amount"`
	real_amount   string `json:"real_Amount"`
	custom        string `json:"custom"`
}

func (that *DbPayment) New() {

	appID := meta.Finance["db"]["app_id"].(string)
	payKey := meta.Finance["db"]["key"].(string)
	apiUrl := meta.Finance["db"]["api"].(string)
	that.Conf = dbPayConf{
		AppID:          appID,
		PayKey:         payKey,
		Name:           "DBPay",
		Domain:         apiUrl,
		PayNotify:      "%s/finance/callback/dbd",
		WithdrawNotify: "%s/finance/callback/dbw",
		Channel: map[string]string{
			"momo":       dbMomo,
			"remit":      dbRemit,
			"online":     dbOnline,
			"viettelpay": dbViettelpay,
		},
	}
}

func (that *DbPayment) Name() string {
	return that.Conf.Name
}

func (that *DbPayment) Pay(orderId, ch, amount, bid string) (paymentDepositResp, error) {

	data := paymentDepositResp{}

	cno, ok := that.Conf.Channel[ch]
	if !ok {
		return data, errors.New(helper.ChannelNotExist)
	}

	now := time.Now()
	recs := map[string]string{
		"uid":        that.Conf.AppID,                                  // 商户编号
		"orderid":    orderId,                                          // 商户订单号
		"channel":    cno,                                              // 纯数字格式; MomoPay:0 | ZaloPay:1 | 银行扫码:2 | 直連:3 | 网关:4 |VTPay:5
		"notify_url": fmt.Sprintf(that.Conf.PayNotify, meta.Fcallback), // 异步通知地址
		"return_url": fmt.Sprintf(that.Conf.PayNotify, meta.Fcallback), // 同步返回地址
		"amount":     fmt.Sprintf("%s000", amount),                     // 订单金额
		"userip":     "86.98.64.30",
		"timestamp":  fmt.Sprintf("%d", now.Unix()), // 时间戳
		"custom":     "",
		"bank_id":    bid, // 银行名称 (用于银行扫码（通道2）,直連（通道3） 的收款账户分配)
	}

	recs["sign"] = that.sign(recs)

	formData := url.Values{}
	for k, v := range recs {
		formData.Set(k, v)
	}
	header := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	uri := fmt.Sprintf("%s/pay", that.Conf.Domain)
	v, err := httpDoTimeout([]byte(formData.Encode()), "POST", uri, header, time.Second*8)
	if err != nil {
		fmt.Println("vnpay uri = ", uri)
		fmt.Println("vnpay httpDoTimeout err = ", err)
		return data, errors.New(helper.PayServerErr)
	}

	var res dbPayResp

	if err = helper.JsonUnmarshal(v, &res); err != nil {
		return data, fmt.Errorf("json format err: %s", err.Error())
	}

	data.Addr = res.Result.Payurl
	data.OrderID = orderId

	return data, nil
}

func (that DbPayment) Withdraw(arg WithdrawAutoParam) (paymentWithdrawalRsp, error) {

	data := paymentWithdrawalRsp{}
	params := map[string]string{
		"uid":           that.Conf.AppID, // 商户编号
		"orderid":       arg.OrderID,     // 商户订单号
		"channel":       "712",
		"notify_url":    fmt.Sprintf(that.Conf.WithdrawNotify, meta.Fcallback), // 异步通知地址
		"amount":        arg.Amount,                                            // 订单金额
		"bank_account":  arg.CardName,                                          // 收款人姓名
		"bank_id":       arg.BankCode,                                          // 收款银行名称
		"bank_no":       arg.CardNumber,                                        // 收款银行账号
		"userip":        "86.98.64.30",                                         //商家会员 ip 地址
		"timestamp":     fmt.Sprintf("%d", arg.Ts.Unix()),                      // 时间戳
		"user_name":     arg.CardName,                                          // 实名
		"bank_province": "",                                                    //开户行所在省份
		"bank_city":     "",                                                    //开户行所在城市
		"bank_sub":      "",                                                    //开户支行
		"custom":        "",
	}

	params["sign"] = that.sign(params)
	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}
	uri := fmt.Sprintf("%s/applyfor", that.Conf.Domain)

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	v, err := httpDoTimeout([]byte(formData.Encode()), "POST", uri, headers, time.Second*8)
	if err != nil {
		return data, err
	}

	var res dbPayWithdrawResp
	err = helper.JsonUnmarshal(v, &res)
	if err != nil {
		return data, err
	}

	data.OrderID = res.Orderid

	return data, nil
}

func (that *DbPayment) PayCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	status := string(fctx.FormValue("status"))
	sign := string(fctx.FormValue("sign"))
	result := fctx.FormValue("result")

	data := paymentCallbackResp{
		State: DepositConfirming,
		Sign:  sign,
	}

	switch status {
	case "10000":
		data.State = DepositSuccess
	case "30901", "30906", "30907", "30911", "30912", "30916", "30921":
		data.State = DepositCancelled
	default:
		return data, fmt.Errorf("unknown status: [%s]", status)
	}

	// check signature
	args := map[string]string{
		"status": status,
		"result": string(result),
	}

	if that.sign(args) != data.Sign {
		return data, fmt.Errorf("invalid sign: { origin: %s , sign: %s, arg: %v} ", data.Sign, that.sign(args), args)
	}

	cbRes := quickDCallBack{}
	err := helper.JsonUnmarshal(result, &cbRes)
	if err != nil {
		return data, fmt.Errorf("parse response error: [%v]", err)
	}

	data.OrderID = cbRes.OrderID
	data.Amount = cbRes.Amount

	return data, nil
}

func (that *DbPayment) WithdrawCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	status := string(fctx.FormValue("status"))
	sign := string(fctx.FormValue("sign"))
	result := fctx.FormValue("result")

	data := paymentCallbackResp{
		State: WithdrawDealing,
		Sign:  sign,
	}

	switch status {
	case "10000":
		data.State = WithdrawSuccess
	case "30901", "30906", "30907", "30911", "30912", "30916", "30921":
		data.State = WithdrawAutoPayFailed
	default:
		return data, fmt.Errorf("unknown status: [%s]", status)
	}

	// check signature
	args := map[string]string{
		"status": status,
		"result": string(fctx.FormValue("result")),
	}

	if that.sign(args) != data.Sign {
		return data, fmt.Errorf("invalid sign")
	}

	cbRes := quickDCallBack{}
	err := helper.JsonUnmarshal(result, &cbRes)
	if err != nil {
		return data, fmt.Errorf("parse response error: [%v]", err)
	}

	data.OrderID = cbRes.OrderID
	data.Amount = cbRes.Amount

	return data, nil
}

func (that *DbPayment) sign(args map[string]string) string {

	i := 0
	qs := ""
	keys := make([]string, len(args))

	for k := range args {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, v := range keys {
		qs += fmt.Sprintf("%s=%s&", v, args[v])
	}
	qs += "key=" + that.Conf.PayKey

	return strings.ToUpper(helper.GetMD5Hash(qs))
}
