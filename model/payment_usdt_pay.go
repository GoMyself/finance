package model

import (
	"finance/contrib/helper"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

type USDTPayment struct {
	Conf USDTConf
}

type USDTConf struct {
	AppID     string
	Name      string
	Domain    string
	Key       string
	PayNotify string
}

type usdtResp struct {
	Status     int          `json:"status"`
	Msg        string       `json:"msg"`
	Data       usdtRespData `json:"data"`
	StatusCode int          `json:"statusCode"`
}

type usdtRespData struct {
	ExchangeRate float64 `json:"exchange_rate"`
	Erc20Address string  `json:"erc20_address"`
	Trc20Address string  `json:"trc20_address"`
	Amount       float64 `json:"amount"`
	UsdtAmount   float64 `json:"usdt_amount"`
	OrderID      string  `json:"order_id"`
}

type resp struct {
	Status int      `json:"status"`
	Msg    int      `json:"msg"`
	Data   respData `json:"data"`
}

type respData struct {
	OrderID string `json:"order_id"`
}

func (that *USDTPayment) New() {

	//appid := "usdt001"
	//key := "xIkkcqTJHoY0zO3fXOHF0QagqSop6439ZvdJratheuBgVTCEnUBE1eiRZKspi4FV"
	//URL := "https://uuu.dabaojian66.xyz/api.html"
	//
	//if meta.IsDev { // 测试账号
	//	appid = "usdt001_test"
	//	key = "7vb9u1vEnwMkCRMIMpksegRyHbKlTTlEtYYvLud2WQ5MZN1cEdTr1UdW"
	//	URL = "http://wallet.selfblock.io/api.html"
	//}

	appid := meta.Finance["USDT"]["app_id"].(string)
	key := meta.Finance["USDT"]["key"].(string)
	URL := meta.Finance["USDT"]["url"].(string)

	that.Conf = USDTConf{
		AppID:     appid,
		Key:       key, // 测试
		Name:      "USDT",
		Domain:    URL,
		PayNotify: "%s/finance/callback/usdtd",
	}
}

func (that *USDTPayment) Name() string {
	return that.Conf.Name
}

func (that *USDTPayment) Pay(orderId, ch, amount, bid string) (paymentDepositResp, error) {

	data := paymentDepositResp{}

	params := map[string]string{
		"order_id":    orderId,                              // 订单号
		"shop_name":   that.Conf.AppID,                      // 商户号
		"method":      "walletpay.create_order",             // 调用的方法
		"time":        fmt.Sprintf("%d", time.Now().Unix()), // 时间戳
		"usdt_amount": fmt.Sprintf("%s", amount),            // 金额
	}

	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}

	formData.Set("access_tonken", that.sign(params))

	uri := fmt.Sprintf("%s?%s", that.Conf.Domain, formData.Encode())
	v, err := httpDoTimeout("usdt", nil, "GET", uri, nil, time.Second*8)
	if err != nil {
		return data, err
	}

	// 处理返回结果
	var rp usdtResp
	if err := helper.JsonUnmarshal(v, &rp); err != nil {
		return data, fmt.Errorf("json format err: %s", err.Error())
	}

	if rp.Status != 1 {
		return data, fmt.Errorf("an 3rd-party error occurred")
	}

	data.Addr = fmt.Sprintf(`{"ERC20": "%s", "TRC20": "%s"}`, rp.Data.Erc20Address, rp.Data.Trc20Address)
	data.OrderID = rp.Data.OrderID
	return data, nil
}

func (that *USDTPayment) Withdraw(arg WithdrawAutoParam) (paymentWithdrawalRsp, error) {

	return paymentWithdrawalRsp{}, nil
}

func (that *USDTPayment) PayCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	//access_tonken	是	string	授权码
	//incharge_id	是	string	交易流水
	//time	是	string	时间戳
	//type	是	string	erc20,trc20
	//usdt_amount	是	string	充值的USDT金额
	//none	是	string	第三发系统的交易单号
	//hash	是	string	区块链单号
	params := map[string]string{}
	fctx.QueryArgs().VisitAll(func(key, value []byte) {
		params[string(key)] = string(value)
	})

	data := paymentCallbackResp{
		State: DepositConfirming,
		Sign:  params["access_tonken"],
	}

	delete(params, "access_tonken")

	if data.Sign != that.sign(params) {
		return data, fmt.Errorf("invalid sign")
	}

	data.State = DepositSuccess

	data.OrderID = params["none"]
	data.Amount = params["usdt_amount"]

	data.Resp = resp{
		Status: 1,
		Msg:    0,
		Data: respData{
			OrderID: params["incharge_id"],
		},
	}
	return data, nil
}

func (that *USDTPayment) WithdrawCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {
	return paymentCallbackResp{}, nil
}

func (that *USDTPayment) sign(args map[string]string) string {

	i := 0
	qs := ""
	keys := make([]string, len(args))

	for k := range args {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, v := range keys {
		qs += fmt.Sprintf("%s|%s", v, args[v])
	}
	qs += that.Conf.Key

	return strings.ToLower(helper.GetMD5Hash(qs))
}
