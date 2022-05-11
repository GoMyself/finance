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
	"github.com/wenzhenxi/gorsa"
)

const (
	jybRemit      = "20"
	jybMomo       = "21"
	jybZalo       = "22"
	jybViettelpay = "23"
	jybOnline     = "24"
)

type jybPayConf struct {
	AppID          string
	PayKey         string
	PaySecret      string
	Publickey      string
	Name           string
	Domain         string
	Key            string
	PayNotify      string
	WithdrawNotify string
	Channel        map[string]string
}

type JybPayment struct {
	Conf jybPayConf
}

type jybPayWithdrawResp struct {
	ResponseContent struct {
		Code      int    `json:"code"`
		Msg       string `json:"msg"`
		Timestamp string `json:"timestamp"`
		Merchno   string `json:"merchno"`
		OrderId   string `json:"orderId"`
		OrderNo   string `json:"orderNo"`
		Status    int    `json:"status"`
	} `json:"responseContent"`
	Sign string `json:"sign"`
}

func (that *JybPayment) New() {

	appID := meta.Finance["jyb"]["app_id"].(string)
	payKey := meta.Finance["jyb"]["key"].(string)
	publickey := meta.Finance["jyb"]["public_key"].(string)
	paySecret := meta.Finance["jyb"]["paySecret"].(string)
	apiUrl := meta.Finance["jyb"]["api"].(string)
	that.Conf = jybPayConf{
		AppID:          appID,
		PayKey:         payKey,
		PaySecret:      paySecret,
		Publickey:      publickey,
		Name:           "jybPay",
		Domain:         apiUrl,
		PayNotify:      "%s/finance/callback/jybd",
		WithdrawNotify: "%s/finance/callback/jybw",
		Channel: map[string]string{
			"remit":      jybRemit,
			"momo":       jybMomo,
			"zalo":       jybZalo,
			"viettelpay": jybViettelpay,
			"online":     jybOnline,
		},
	}
}

func (that *JybPayment) Name() string {
	return that.Conf.Name
}

func (that *JybPayment) Pay(orderId, ch, amount, bid string) (paymentDepositResp, error) {

	data := paymentDepositResp{}

	cno, ok := that.Conf.Channel[ch]
	if !ok {
		return data, errors.New(helper.ChannelNotExist)
	}

	now := time.Now()
	recs := map[string]string{
		"merchno":         that.Conf.AppID, // 商户编号
		"orderId":         orderId,         // 商户订单号
		"payType":         cno,             // 纯数字格式; MomoPay:0 | ZaloPay:1 | 银行扫码:2 | 直連:3 | 网关:4 |VTPay:5
		"requestCurrency": "3",
		"amount":          fmt.Sprintf("%s000.00", amount),                  // 订单金额
		"requestTime":     now.Format("20060102150405"),                     // 日期时间 (格式:2018-01-01 23:59:59)
		"asyncUrl":        fmt.Sprintf(that.Conf.PayNotify, meta.Fcallback), // 异步通知地址
		"syncUrl":         fmt.Sprintf(that.Conf.PayNotify, meta.Fcallback), // 异步通知地址
		"apiVersion":      "2",                                              //
	}
	if bid != "" {
		recs["bankCode"] = bid
	}
	recs["sign"] = that.sign(recs, "deposit")

	//formData := url.Values{}
	//for k, v := range recs {
	//	formData.Set(k, v)
	//}
	//
	//uri := fmt.Sprintf("%s/api/order/placeOrder?%s", that.Conf.Domain, paramEncode(recs))
	//fmt.Println(uri)
	//data.Addr = uri
	//data.OrderID = log.OrderID
	//
	//return data, nil

	//formData := url.Values{}
	//for k, v := range recs {
	//	formData.Set(k, v)
	//}
	//header := map[string]string{
	//	"Content-Type": "application/x-www-form-urlencoded",
	//}

	uri := fmt.Sprintf("%s/api/order/placeOrder", that.Conf.Domain)
	//v, err := httpDoTimeout([]byte(formData.Encode()), "POST", uri, header, time.Second*8, log)
	//if err != nil {
	//	return data, err
	//}
	data.Addr = uri
	data.OrderID = orderId
	data.IsForm = "1"
	for k, v := range recs {
		data.Data[k] = v
	}

	return data, nil
}

func (that *JybPayment) Withdraw(arg WithdrawAutoParam) (paymentWithdrawalRsp, error) {

	data := paymentWithdrawalRsp{}
	params := map[string]string{
		"merchno":         that.Conf.AppID,                                       // 商户编号
		"orderId":         arg.OrderID,                                           // 商户订单号
		"amount":          fmt.Sprintf("%s.00", arg.Amount),                      // 订单金额
		"tradeType":       "1",                                                   //1：对私；2：对公；（目前只支持对私交易）
		"account":         arg.CardName,                                          // 收款人姓名
		"cardNo":          arg.CardNumber,                                        // 收款银行账号
		"bankName":        arg.BankCode,                                          // 收款银行名称
		"asyncUrl":        fmt.Sprintf(that.Conf.WithdrawNotify, meta.Fcallback), // 异步通知地址
		"timestamp":       arg.Ts.Format("20060102150405"),                       // 日期时间 (格式:yyyyMMddHHmmss)
		"cashType":        "3",                                                   //下发类型(下发通道)1：人民币；2：USDT；3：越南盾；4：印度卢比
		"requestCurrency": "3",                                                   //请求货币1：人民币；2：USDT；3：越南盾；4：印度卢比
		"apiVersion":      "2",
	}

	params["sign"] = that.sign(params, "withdraw")
	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}
	uri := fmt.Sprintf("%s/api/cash/queryCash", that.Conf.Domain)

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	v, err := httpDoTimeout([]byte(formData.Encode()), "POST", uri, headers, time.Second*8)
	if err != nil {
		return data, err
	}

	var res jybPayWithdrawResp
	err = helper.JsonUnmarshal(v, &res)
	if err != nil {
		return data, err
	}

	if res.ResponseContent.Code != 0 {
		return data, fmt.Errorf("an 3rd-party error occurred")
	}

	data.OrderID = res.ResponseContent.OrderId

	return data, nil
}

func (that *JybPayment) PayCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	params := make(map[string]string)
	fctx.PostArgs().VisitAll(func(key, value []byte) {
		params[string(key)] = string(value)
	})

	data := paymentCallbackResp{
		State: DepositConfirming,
		Sign:  string(fctx.PostArgs().Peek("sign")),
	}

	if !valid(params, []string{"merchno", "orderId", "amount", "requestCurrency", "payType", "apiVersion", "status", "sign"}) {
		return data, fmt.Errorf("param err: [%v]", params)
	}

	switch params["status"] {
	case "2":
		data.State = DepositSuccess
	default:
		return data, fmt.Errorf("unknown status: [%s]", params["status"])
	}

	if that.sign(params, "deposit") != params["sign"] {
		return data, fmt.Errorf("invalid sign")
	}

	data.OrderID = params["orderId"]
	data.Amount = params["amount"]

	return data, nil
}

func (that *JybPayment) WithdrawCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	params := make(map[string]string)
	fctx.PostArgs().VisitAll(func(key, value []byte) {
		params[string(key)] = string(value)
	})

	data := paymentCallbackResp{
		State: WithdrawDealing,
		Sign:  string(fctx.PostArgs().Peek("sign")),
	}

	if !valid(params, []string{"timestamp", "orderNo", "merchno", "orderId", "amount", "tradeType", "account", "cardNo", "bankName",
		"cashType", "requestCurrency", "apiVersion", "sign", "status"}) {
		return data, fmt.Errorf("param err: [%v]", params)
	}

	switch params["status"] {
	case "1":
		data.State = WithdrawSuccess
	case "2":
		data.State = WithdrawAutoPayFailed
	default:
		return data, fmt.Errorf("unknown status: [%s]", params["status"])
	}

	if that.sign(params, "withdraw") != data.Sign {
		return data, fmt.Errorf("invalid sign")
	}

	data.OrderID = params["orderId"]
	data.Amount = params["amount"]

	return data, nil
}

func (that *JybPayment) sign(args map[string]string, method string) string {

	qs := ""
	keys := make([]string, 0)

	for k := range args {
		keys = append(keys, k)
	}

	if method == "deposit" {
		sort.Strings(keys)
		for _, v := range keys {
			qs += fmt.Sprintf("%s=%s&", v, args[v])
		}
		qs = qs + "secretKey=" + that.Conf.PayKey
		fmt.Println(qs)
		return strings.ToLower(helper.GetMD5Hash(qs))
	}

	if method == "withdraw" {
		sort.Strings(keys)
		for _, v := range keys {
			qs += fmt.Sprintf("%s=%s&", v, args[v])
		}
		qs = qs + "secretKey=" + that.Conf.PayKey

		fmt.Println(qs)
		sign, err := gorsa.PriKeyEncrypt(strings.ToLower(helper.GetMD5Hash(qs)), strings.ReplaceAll(that.Conf.PaySecret, "\n", ""))
		if err != nil {
			fmt.Println(err)
			return ""
		}
		fmt.Println(sign)
		return sign
	}
	return ""
}
