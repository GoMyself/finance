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

//QuickPayment quick支付
type QuickPayment struct {
	Conf quickConf
}

type quickPayResp struct {
	Status int         `json:"status"`
	Result quickResult `json:"result"`
}

type quickResult struct {
	TransactionID int64  `json:"transactionid,string"`
	PayURL        string `json:"payurl"`
	Points        string `json:"points"`
}

type quickDCallBack struct {
	OrderID string `json:"orderid"`
	Amount  string `json:"amount,float64"`
}

type quickWithdrawResp struct {
	Status int         `json:"status"`
	Result quickResult `json:"result"`
	Sign   string      `json:"sign"`
}

type quickConf struct {
	AppID          string
	Name           string
	Domain         string
	Key            string
	PayNotify      string
	PayReturn      string
	WithdrawNotify string
	Channel        map[string]string
}

const (
	quickMomo     = "923" // momo
	quickZalo     = "921" // zalo
	quickOnline   = "907" // online
	quickUnionpay = "908" // unionpay
)

const (
	quickSuccess      = "10000" // 成功
	quickValid        = "30911" // 验证失败
	quickRealName     = "30912" // 实名失败
	quickTrade        = "30916" // 交易失败
	quickTimeOut      = "30921" // 交易超时
	quickOrderExpired = "30901" // 登入失败
	quickLogin        = "30906" // 验证失败
	quickAmount       = "30907" // 余额不足

)

//New 初始化配置信息
func (that *QuickPayment) New() {

	//appID := "kokvn"
	//key := "2PKUEBNSRdrNoPsW"
	//
	//if meta.IsDev { // 测试账号
	//	appID = "baoer"
	//	key = "oaX2ffgsiDonwfAY"
	//}

	appID := meta.Finance["quick"]["app_id"].(string)
	key := meta.Finance["quick"]["key"].(string)

	that.Conf = quickConf{
		AppID:          appID,
		Key:            key,
		Name:           "QuickPay",
		Domain:         "https://api.quickpay.support",
		PayNotify:      "%s/finance/callback/quickd",
		PayReturn:      "",
		WithdrawNotify: "%s/finance/callback/quickw",
		Channel: map[string]string{
			"momo":     quickMomo,     // momo
			"zalo":     quickZalo,     // zalo
			"online":   quickOnline,   // online
			"unionpay": quickUnionpay, // unionpay
		},
	}
}

//Name 支付名称
func (that *QuickPayment) Name() string {
	return that.Conf.Name
}

//Pay 发起支付
func (that *QuickPayment) Pay(log *paymentTDLog, ch, amount, bid string) (paymentDepositResp, error) {

	data := paymentDepositResp{}

	cno, ok := that.Conf.Channel[ch]
	if !ok {
		return data, errors.New(helper.ChannelNotExist)
	}

	params := map[string]string{
		"uid":        that.Conf.AppID,                                  // 商户 ID
		"orderid":    log.OrderID,                                      // 订单号
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

	uri := fmt.Sprintf("%s/pay", that.Conf.Domain)
	headers := map[string]string{}

	res, err := httpDoTimeout([]byte(formData.Encode()), "POST", uri, headers, time.Second*8, log)
	if err != nil {
		return data, err
	}

	var rp quickPayResp
	if err := helper.JsonUnmarshal(res, &rp); err != nil {
		return data, fmt.Errorf("json format err: %s", err.Error())
	}

	if rp.Status != 10000 {
		return data, fmt.Errorf("an 3rd-party error occurred")
	}

	data.Addr = rp.Result.PayURL
	data.OrderID = strconv.FormatInt(rp.Result.TransactionID, 10)
	return data, nil
}

//PayCallBack 支付回调
func (that *QuickPayment) PayCallBack(ctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	status := string(ctx.FormValue("status"))
	sign := string(ctx.FormValue("sign"))
	result := ctx.FormValue("result")

	data := paymentCallbackResp{
		State: DepositConfirming,
		Sign:  sign,
	}

	switch status {
	case quickSuccess:
		data.State = DepositSuccess
	case quickValid, quickTrade, quickLogin,
		quickAmount, quickTimeOut, quickRealName, quickOrderExpired:
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

//Withdraw 发起代付
func (that *QuickPayment) Withdraw(log *paymentTDLog, arg WithdrawAutoParam) (paymentWithdrawalRsp, error) {

	data := paymentWithdrawalRsp{}
	params := map[string]string{
		"uid":           that.Conf.AppID,                                       // 商户 ID
		"orderid":       arg.OrderID,                                           // 订单号
		"channel":       "712",                                                 // 代付类型 712越南银行代付 713印度银行代付
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

	v, err := httpDoTimeout([]byte(formData.Encode()), "POST", uri, headers, time.Second*8, log)
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

//WithdrawCallBack 代付回调
func (that *QuickPayment) WithdrawCallBack(ctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	status := string(ctx.FormValue("status"))
	sign := string(ctx.FormValue("sign"))
	result := ctx.FormValue("result")

	data := paymentCallbackResp{
		State: WithdrawDealing,
		Sign:  sign,
	}

	switch status {
	case quickSuccess:
		data.State = WithdrawSuccess
	case quickValid, quickTrade, quickLogin,
		quickAmount, quickTimeOut, quickRealName, quickOrderExpired:
		data.State = WithdrawAutoPayFailed
	default:
		return data, fmt.Errorf("unknown status: [%s]", status)
	}

	// check signature
	args := map[string]string{
		"status": status,
		"result": string(result),
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

func (that *QuickPayment) sign(args map[string]string) string {

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
