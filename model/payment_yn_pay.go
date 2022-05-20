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

	"github.com/shopspring/decimal"
	"github.com/valyala/fasthttp"
)

type ynConf struct {
	AppID          string
	PayKey         string
	Password       string
	Name           string
	Domain         string
	PayNotify      string
	WithdrawNotify string
	Channel        map[string]string
}

type YNPayment struct {
	Conf ynConf
}

type ynPayResp struct {
	Code int `json:"code"`
	Data struct {
		BankName string `json:"bankName"`
		Card     string `json:"card"`
		Name     string `json:"name"`
		PayUrl   string `json:"payUrl"`
		Sn       string `json:"sn"`
	} `json:"data"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}

type ynWithdrawResp struct {
	Code int `json:"code"`
	Data struct {
		Sn string `json:"sn"`
	} `json:"data"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}

type ynPayCallbackBody struct {
	Amount       float64 `json:"amount"`
	PayTime      string  `json:"payTime"`
	ActualAmount float64 `json:"actualAmount"`
	OutTradeNo   string  `json:"outTradeNo"`
	Sn           string  `json:"sn"`
	Status       string  `json:"status"`
}

func (that *YNPayment) New() {

	//appID := "202202101844036748"
	//payKey := "37d3463469e543c9abbff84ee029d0f2"
	//password := "ck718293"
	//
	//if meta.IsDev {
	//	appID = "201909151500453813"
	//	payKey = "44966078044c403cb834804e6ac94373"
	//	password = "test123123"
	//}

	appID := meta.Finance["yn"]["app_id"].(string)
	payKey := meta.Finance["yn"]["key"].(string)
	password := meta.Finance["yn"]["password"].(string)

	that.Conf = ynConf{
		AppID:          appID,
		PayKey:         payKey,
		Password:       password,
		Name:           "YN",
		Domain:         "http://18.163.8.208:8083",
		PayNotify:      "%s/finance/callback/ynd",
		WithdrawNotify: "%s/finance/callback/ynw",
		Channel: map[string]string{
			"auto":     "ZK",   // 复制转卡
			"unionpay": "QRZK", // 扫码转卡
		},
	}
}

func (that *YNPayment) Name() string {
	return that.Conf.Name
}

func (that *YNPayment) Pay(orderId, ch, amount, bid string) (paymentDepositResp, error) {

	data := paymentDepositResp{}

	cno, ok := that.Conf.Channel[ch]
	if !ok {
		return data, errors.New(helper.ChannelNotExist)
	}

	now := time.Now()
	recs := map[string]string{
		"appId":      that.Conf.AppID,                                  // 商户编号
		"outTradeNo": orderId,                                          // 商户订单号
		"payType":    cno,                                              // 1-KB (复制模式) 2-FX(飞行模式) 3-SC(扫码模式) 4-SQ(自动启动)
		"amount":     fmt.Sprintf("%s000.00", amount),                  // 订单金额
		"asyncUrl":   fmt.Sprintf(that.Conf.PayNotify, meta.Fcallback), // 异步通知地址
		"nonceStr":   fmt.Sprintf("%d", now.Unix()),                    // 时间戳
		"ip":         "203.208.43.98",                                  //
	}

	if cno == "QRZK" {
		recs["bankCode"] = bid
	}

	singStr := ""
	singStr, recs["sign"] = that.sign(recs, "deposit")
	fmt.Printf("singStr:%s,\n sign:%s \n", singStr, recs["sign"])
	formData := url.Values{}
	for k, v := range recs {
		formData.Set(k, v)
	}

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	uri := fmt.Sprintf("%s/api/v1/pay", that.Conf.Domain)
	v, err := httpDoTimeout("yn", []byte(formData.Encode()), "POST", uri, headers, time.Second*8)
	if err != nil {
		fmt.Println("yn uri = ", uri)
		fmt.Println("yn httpDoTimeout err = ", err)

		return data, errors.New(helper.PayServerErr)
	}

	var res ynPayResp
	if err = helper.JsonUnmarshal(v, &res); err != nil {
		return data, fmt.Errorf("json format err: %s", err.Error())
	}

	if res.Code != 200 {
		return data, fmt.Errorf("an 3rd-party error occurred")
	}

	data.Addr = res.Data.PayUrl
	data.OrderID = res.Data.Sn

	return data, nil
}

func (that *YNPayment) Withdraw(arg WithdrawAutoParam) (paymentWithdrawalRsp, error) {

	data := paymentWithdrawalRsp{}

	params := map[string]string{
		"appId":      that.Conf.AppID,                                       // 商户 ID
		"outTradeNo": arg.OrderID,                                           // 订单号
		"asyncUrl":   fmt.Sprintf(that.Conf.WithdrawNotify, meta.Fcallback), // 异步通知地址
		"amount":     arg.Amount,                                            // 金额
		"nonceStr":   strconv.FormatInt(arg.Ts.Unix(), 10),                  // 时间戳
		"name":       arg.CardName,                                          // 收款姓名
		"card":       arg.CardNumber,                                        // 收款卡号
		"bankBranch": arg.BankAddress,                                       // 收款支行
		"bankCode":   arg.BankCode,                                          // 收款编码
		"password":   helper.MD5Hash(that.Conf.Password),
	}

	_, params["sign"] = that.sign(params, "withdraw")
	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}
	uri := fmt.Sprintf("%s/api/v1/issued", that.Conf.Domain)
	headers := map[string]string{}

	v, err := httpDoTimeout("yn", []byte(formData.Encode()), "POST", uri, headers, time.Second*8)
	if err != nil {
		return data, err
	}

	var res ynWithdrawResp
	err = helper.JsonUnmarshal(v, &res)
	if err != nil {
		return data, err
	}

	if !res.Success {
		return data, fmt.Errorf("an 3rd-party error occurred: %s", string(v))
	}

	data.OrderID = res.Data.Sn

	return data, nil
}

func (that *YNPayment) PayCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	params := make(map[string]string)
	fctx.PostArgs().VisitAll(func(key, value []byte) {
		params[string(key)] = string(value)
	})

	data := paymentCallbackResp{
		State: DepositConfirming,
		Sign:  string(fctx.FormValue("sign")),
	}

	var body ynPayCallbackBody
	err := helper.JsonUnmarshal([]byte(params["body"]), &body)
	if err != nil {
		return data, fmt.Errorf("param err: [%v]", params)
	}

	if !valid(params, []string{"sign", "body"}) {
		return data, fmt.Errorf("param err: [%v]", params)
	}

	switch body.Status {
	case "success", "work": // work 补单
		data.State = DepositSuccess
	case "failure":
		data.State = DepositCancelled
	default:
		return data, fmt.Errorf("unknown status: [%s]", body.Status)
	}

	if that.backSign(params["body"]) != params["sign"] {
		return data, fmt.Errorf("invalid sign")
	}

	data.OrderID = body.OutTradeNo
	data.Amount = decimal.NewFromFloat(body.Amount).Truncate(2).String()

	return data, nil
}

func (that *YNPayment) WithdrawCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	params := make(map[string]string)
	fctx.PostArgs().VisitAll(func(key, value []byte) {
		if "body" == string(key) {
			body, err := url.QueryUnescape(string(value))
			if err != nil {
				fmt.Println("yn withdraw url decode err", err.Error())
				return
			}
			params[string(key)] = body
			return
		}
		params[string(key)] = string(value)
	})

	data := paymentCallbackResp{
		State: DepositConfirming,
		Sign:  string(fctx.FormValue("sign")),
	}

	var body ynPayCallbackBody
	err := helper.JsonUnmarshal([]byte(params["body"]), &body)
	if err != nil {
		return data, fmt.Errorf("param err: [%v]", params)
	}

	if !valid(params, []string{"sign", "body"}) {
		return data, fmt.Errorf("param err: [%v]", params)
	}

	switch body.Status {
	case "success": // work 补单
		data.State = WithdrawSuccess
	case "failure":
		data.State = WithdrawAutoPayFailed
	default:
		return data, fmt.Errorf("unknown status: [%s]", body.Status)
	}

	if that.backSign(params["body"]) != params["sign"] {
		return data, fmt.Errorf("invalid sign")
	}

	data.OrderID = body.OutTradeNo
	data.Amount = decimal.NewFromFloat(body.Amount).Truncate(2).String()

	return data, nil
}

func (that *YNPayment) sign(args map[string]string, ty string) (string, string) {

	qs := ""
	keys := make([]string, 0)

	for k, v := range args {

		if k == "sign" || v == "" {
			continue
		}

		if ty == "deposit" && k == "bankCode" {
			continue
		}

		keys = append(keys, k)
	}

	sort.Strings(keys)
	for _, v := range keys {
		qs += fmt.Sprintf("%s=%s&", v, args[v])
	}

	qs += "key=" + that.Conf.PayKey
	return qs, strings.ToUpper(helper.GetMD5Hash(qs))
}

func (that *YNPayment) backSign(body string) string {
	qs := fmt.Sprintf("body=%s&key=%s", body, that.Conf.PayKey)
	return strings.ToUpper(helper.GetMD5Hash(qs))
}
