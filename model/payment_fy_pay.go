package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

const (
	fyMomo     = "923"
	fyZalo     = "921"
	fyOnline   = "907"
	fyUnionPay = "908"
)

type FyPayment struct {
	Conf fyConf
}

type fyConf struct {
	AppID          string
	Name           string
	Domain         string
	Key            string
	PayNotify      string
	PayReturn      string
	WithdrawNotify string
	Channel        map[string]string
}

func (that *FyPayment) New() {

	//appID := "KOKVN"
	//key := "FpG9QFd6LDiBXuMD"
	//
	//if meta.IsDev { // 测试账号
	//	appID = "KOK02"
	//	key = "Bre2B23Yvw4n1LIi"
	//}
	appID := meta.Finance["fy"]["app_id"].(string)
	key := meta.Finance["fy"]["key"].(string)
	that.Conf = fyConf{
		AppID:          appID, // 测试
		Key:            key,   // 测试
		Name:           "FY",
		Domain:         "https://api.fy13.support",
		PayNotify:      "%s/finance/callback/fyd",
		PayReturn:      "",
		WithdrawNotify: "%s/finance/callback/fyw",
		Channel: map[string]string{
			"momo":     fyMomo,     // momo
			"zalo":     fyZalo,     // zalo
			"online":   fyOnline,   // online
			"unionpay": fyUnionPay, // unionPay
		},
	}
}

func (that *FyPayment) Name() string {
	return that.Conf.Name
}

func (that *FyPayment) Pay(orderId, ch, amount, bid string) (paymentDepositResp, error) {

	data := paymentDepositResp{}

	cno, ok := that.Conf.Channel[ch]
	if !ok {
		return data, errors.New(helper.ChannelNotExist)
	}

	params := map[string]string{
		"uid":        that.Conf.AppID,                                  // 商户 ID
		"orderid":    orderId,                                          // 订单号
		"channel":    cno,                                              // 支付类型
		"notify_url": fmt.Sprintf(that.Conf.PayNotify, meta.Fcallback), // 异步通知地址
		"return_url": fmt.Sprintf(that.Conf.PayNotify, meta.Fcallback), // 同步返回地址
		"amount":     fmt.Sprintf("%s000", amount),                     // 金额
		"userip":     "203.208.43.98",                                  // 客端 IP
		"timestamp":  fmt.Sprintf("%d", time.Now().Unix()),             // 时间戳
		"custom":     "",                                               // 自定义
	}

	if bid != "" {
		params["bank_id"] = bid // 银行编号
	}

	params["sign"] = that.sign(params)

	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}

	headers := map[string]string{}
	uri := fmt.Sprintf("%s/pay", that.Conf.Domain)

	v, err := httpDoTimeout("fy pay", []byte(formData.Encode()), "POST", uri, headers, time.Second*8)
	if err != nil {
		fmt.Println("fy uri = ", uri)
		fmt.Println("fy httpDoTimeout err = ", err)
		return data, errors.New(helper.PayServerErr)
	}

	// 处理返回结果
	var rp quickPayResp
	if err := helper.JsonUnmarshal(v, &rp); err != nil {
		return data, fmt.Errorf("json format err: %s", err.Error())
	}

	if rp.Status != 10000 {
		return data, fmt.Errorf("an 3rd-party error occurred")
	}

	data.Addr = rp.Result.PayURL
	data.OrderID = strconv.FormatInt(rp.Result.TransactionID, 10)
	return data, nil
}

func (that *FyPayment) Withdraw(arg WithdrawAutoParam) (paymentWithdrawalRsp, error) {

	data := paymentWithdrawalRsp{}
	params := map[string]string{
		"uid":           that.Conf.AppID,                                       // 商户 ID
		"orderid":       arg.OrderID,                                           // 订单号
		"channel":       "712",                                                 // 支付类型
		"notify_url":    fmt.Sprintf(that.Conf.WithdrawNotify, meta.Fcallback), // 异步通知地址
		"amount":        arg.Amount,                                            // 金额
		"userip":        "203.208.43.98",                                       // 客端 IP
		"timestamp":     fmt.Sprintf("%d", arg.Ts.Unix()),                      // 时间戳
		"custom":        "",                                                    // 自定义
		"user_name":     arg.CardName,                                          // 实名
		"bank_account":  arg.CardName,                                          // 收款人开户姓名
		"bank_no":       arg.CardNumber,                                        // 收款人银行帐号
		"bank_id":       arg.BankCode,                                          // 银行编号
		"bank_province": "",                                                    // 开户行所在省份
		"bank_city":     "",                                                    // 开户行所在城市
		"bank_sub":      "",                                                    // 开户支行
	}

	params["sign"] = that.sign(params)
	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}
	uri := fmt.Sprintf("%s/applyfor", that.Conf.Domain)
	headers := map[string]string{}

	v, err := httpDoTimeout("fy pay", []byte(formData.Encode()), "POST", uri, headers, time.Second*8)
	if err != nil {
		return data, err
	}

	var res quickWithdrawResp
	err = helper.JsonUnmarshal(v, &res)
	if err != nil {
		return data, err
	}

	if res.Status != 10000 {
		return data, fmt.Errorf("an 3rd-party error occurred")
	}

	data.OrderID = strconv.FormatInt(res.Result.TransactionID, 10)

	return data, nil
}

func (that *FyPayment) PayCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

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

func (that *FyPayment) WithdrawCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

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

func (that *FyPayment) sign(args map[string]string) string {

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
	qs += "key=" + that.Conf.Key

	return strings.ToUpper(helper.GetMD5Hash(qs))
}
