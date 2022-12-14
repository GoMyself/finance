package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
)

const (
	p3Online     = "online"
	p3Offline    = "offline"
	p3QR         = "qr"
	p3MOMO       = "momo"
	p3ZALO       = "zalo"
	p3VIETTELPAY = "viettelpay"
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
	Channel        map[string]string
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

type vnPayCallBack struct {
	MerchantNo      string `json:"merchantNo"`      //商户号
	MerchantOrderNo string `json:"merchantOrderNo"` // 订单号
	ChannelCode     string `json:"channelCode"`     // 充值(collection) or 提现(withdraw)
	OrderNo         string `json:"orderNo"`         // verified = 已完成 & revoked = 被撒销 timeout = 逾时 & processing = 處理中
	Currency        string `json:"currency"`        // 币种
	Amount          string `json:"amount"`          // 订单金额
	UserId          string `json:"userId"`
	Extra           string `json:"extra"`
	Status          string `json:"status"`
	Sign            string `json:"sign"`
}

type qrPayResp struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		OrderNo      string `json:"orderNo"`
		Account      string `json:"account"`      //收款账号
		Name         string `json:"name"`         //收款人姓名
		QrCodeUrl    string `json:"qrCodeUrl"`    //二维码
		Amount       string `json:"amount"`       //支付金额
		PayCode      string `json:"payCode"`      //收款确认码
		BankCode     string `json:"bankCode"`     //收款银行编码
		UserCode     string `json:"userCode"`     //付款银行编码(用户选择的银行)
		PayResult    string `json:"payResult"`    //充值结果，用于通知用户充值结果。Create,创建。Success，成功
		PayInPicH5   string `json:"payInPicH5"`   //收款银行图片h5
		PayInPicWeb  string `json:"payInPicWeb"`  //收款银行图片web
		PayOutPicH5  string `json:"payOutPicH5"`  //付款银行图片H5
		PayOutPicWeb string `json:"payOutPicWeb"` //付款银行图片H5
		Style        string `json:"style"`        //模板样式 1：样式1，2：样式2，3：样式3
		EndSecond    uint64 `json:"endSecond"`    //倒计时，通过请求计算
		StartDate    string `json:"startDate"`    //支付计时时间

	} `json:"data"`
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
		Name:           "P3Pay",
		Domain:         apiUrl,
		PayNotify:      "%s/finance/callback/vnd",
		WithdrawNotify: "%s/finance/callback/vnw",
		Channel: map[string]string{
			"3": p3Online,
			"9": p3Offline,
			"1": p3MOMO,
			"8": p3QR,
			"2": p3ZALO,
			"6": p3VIETTELPAY,
		},
	}
}

func (that *VnPayment) Name() string {
	return that.Conf.Name
}

func (that *VnPayment) Pay(orderId, ch, amount, bid string) (paymentDepositResp, error) {

	data := paymentDepositResp{}
	fmt.Println("vnpay ch = ", ch)
	cno, ok := that.Conf.Channel[ch]
	if !ok {
		return data, errors.New(helper.ChannelNotExist)
	}
	var res vnPayResp
	var path string

	now := time.Now()
	recs := map[string]string{
		"merchantNo":  that.Conf.MerchanNo,                              // 商户编号
		"channelCode": bid,                                              // 银行名称 (用于银行扫码（通道2）,直連（通道3） 的收款账户分配)
		"orderNo":     orderId,                                          // 商户订单号
		"currency":    "VND",                                            //
		"amount":      fmt.Sprintf("%s000", amount),                     // 订单金额
		"notifyUrl":   fmt.Sprintf(that.Conf.PayNotify, meta.Fcallback), // 异步通知地址
		"targetUrl":   meta.IndexUrl,
		"versionStr":  "new",
	}
	if cno == p3Online || cno == p3Offline {
		recs["bankDirct"] = cno
	}

	tp := fmt.Sprintf("%d", now.UnixMilli())
	recs["timestamp"] = tp
	if cno == p3MOMO {
		recs["channelCode"] = "MOMO"
	}
	if cno == p3ZALO {
		recs["channelCode"] = "ZALO"
	}
	if cno == p3VIETTELPAY {
		recs["channelCode"] = "VPAY"
	}
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
	if cno == p3Online {
		path = "/v1/api/online/ebank/"
	}
	if cno == p3Offline {
		path = "/v1/api/offline/deposit/"
	}
	if cno == p3QR || cno == p3MOMO || cno == p3ZALO || cno == p3VIETTELPAY {
		path = "/v1/api/pay/scan/"
	}

	uri := fmt.Sprintf("%s%s%s/%s/%s", that.Conf.Domain, path, that.Conf.AppID, that.Conf.Merchan, orderId)

	v, err := httpDoTimeout("p3 pay", body, "POST", uri, header, time.Second*8)
	if err != nil {
		fmt.Println("vnpay uri = ", uri)
		fmt.Println("vnpay httpDoTimeout err = ", err)
		fmt.Println("vnpay body = ", string(v))
		return data, errors.New(helper.PayServerErr)
	}
	fmt.Println("vnpay body = ", string(v))
	if err = helper.JsonUnmarshal(v, &res); err != nil {
		return data, fmt.Errorf("json format err: %s", err.Error())
	}

	if res.Code != "0000" {
		return data, fmt.Errorf("an 3rd-party error occurred")
	}

	data.Addr = res.Data.Link
	data.OrderID = res.Data.OrderNo
	data.UseLink = 1

	return data, nil
}

func (that *VnPayment) Withdraw(arg WithdrawAutoParam) (paymentWithdrawalRsp, error) {

	data := paymentWithdrawalRsp{}
	params := map[string]string{
		"merchantNo":    that.Conf.MerchanNo, // 商户编号
		"channelCode":   arg.BankCode,        // 收款银行名称
		"orderNo":       arg.OrderID,         // 商户订单号
		"currency":      "VND",
		"amount":        fmt.Sprintf("%s", arg.Amount),                         // 订单金额
		"payee":         arg.CardName,                                          // 收款人姓名
		"payeeBankCard": arg.CardNumber,                                        // 收款银行账号
		"notifyUrl":     fmt.Sprintf(that.Conf.WithdrawNotify, meta.Fcallback), // 异步通知地址
		"verifyUrl":     "",                                                    // 验证订单地址,若提供则,我方 post 请 求验证,默认返回 {“code”:”0000”}
	}
	now := time.Now()
	tp := fmt.Sprintf("%d", now.UnixMilli())
	params["timestamp"] = tp
	params["sign"] = that.sign(params, "withdraw")
	delete(params, "timestamp")
	body, err := helper.JsonMarshal(params)
	if err != nil {
		return data, errors.New(helper.FormatErr)
	}
	sid := helper.GenId()
	uri := fmt.Sprintf("%s/v1/api/withdraw/%s/%s/%s", that.Conf.Domain, that.Conf.AppID, that.Conf.Merchan, arg.OrderID)
	header := map[string]string{
		"Content-Type": "application/json",
		"Nonce":        helper.MD5Hash(sid),
		"Timestamp":    tp,
		"x-Request-Id": sid,
	}
	v, err := httpDoTimeout("p3 pay", body, "POST", uri, header, time.Second*8)
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

func (that *VnPayment) PayCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	data := paymentCallbackResp{
		State: DepositConfirming,
	}

	var p fastjson.Parser
	v, err := p.ParseBytes(fctx.PostBody())
	if err != nil {
		fmt.Println("PayCallBack content error : ", err, string(fctx.PostBody()))
	}
	fmt.Println("v:", v.String())
	params := vnPayCallBack{}
	if err := helper.JsonUnmarshal(fctx.PostBody(), &params); err != nil {
		return data, fmt.Errorf("param format err: %s", err.Error())
	}
	fmt.Println("params", params)

	data.Sign = params.Sign

	switch params.Status {
	case "Success":
		data.State = DepositSuccess
	default:
		return data, fmt.Errorf("unknown status: [%s]", params.Status)
	}

	paraMap := map[string]string{
		"merchantNo":      that.Conf.MerchanNo,
		"merchantOrderNo": params.MerchantOrderNo,
		"orderNo":         params.OrderNo,
		"amount":          params.Amount,
		"status":          params.Status,
	}
	if that.sign(paraMap, "call") != params.Sign {
		return data, fmt.Errorf("invalid sign")
	}

	data.OrderID = params.OrderNo
	data.Amount = params.Amount
	resp := &vnPayWithdrawResp{
		Code: "0000",
		Msg:  "Success",
		Data: params.OrderNo,
	}
	data.Resp = resp
	return data, nil
}

func (that *VnPayment) WithdrawCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	data := paymentCallbackResp{
		State: WithdrawDealing,
	}
	params := vnPayCallBack{}
	if err := helper.JsonUnmarshal(fctx.PostBody(), &params); err != nil {
		return data, fmt.Errorf("param format err: %s", err.Error())
	}

	fmt.Println(params)

	data.Sign = params.Sign

	switch params.Status {
	case "Success":
		data.State = WithdrawSuccess
	case "Failure":
		data.State = WithdrawAutoPayFailed
	default:
		return data, fmt.Errorf("unknown status: [%s]", params.Status)
	}

	paraMap := map[string]string{
		"merchantNo":      params.MerchantNo,
		"merchantOrderNo": params.MerchantOrderNo,
		"orderNo":         params.OrderNo,
		"channelCode":     params.ChannelCode,
		"currency":        params.Currency,
		"amount":          params.Amount,
		"status":          params.Status,
	}
	if that.sign(paraMap, "withdrawcall") != data.Sign {
		return data, fmt.Errorf("invalid sign")
	}

	data.OrderID = params.MerchantOrderNo
	data.Amount = params.Amount
	resp := &vnPayWithdrawResp{
		Code: "0000",
		Msg:  "Success",
		Data: params.MerchantOrderNo,
	}
	data.Resp = resp
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
		qs += fmt.Sprintf(`merchantNo=%s&orderNo=%s&merchantOrderNo=%s&amount=%s&status=%s`,
			args["merchantNo"], args["orderNo"], args["merchantOrderNo"], args["amount"],
			args["status"])
	}

	if method == "withdraw" {
		qs += fmt.Sprintf(`merchantNo=%s&channelCode=%s&orderNo=%s&currency=%s&amount=%s&payee=%s&payeeBankCard=%s&notifyUrl=%s&timestamp=%s`,
			args["merchantNo"], args["channelCode"], args["orderNo"], args["currency"], args["amount"], args["payee"],
			args["payeeBankCard"], args["notifyUrl"], args["timestamp"])
	}

	if method == "withdrawcall" {
		qs += fmt.Sprintf(`merchantNo=%s&merchantOrderNo=%s&orderNo=%s&channelCode=%s&currency=%s&amount=%s&status=%s`,
			args["merchantNo"], args["merchantOrderNo"], args["orderNo"], args["channelCode"],
			args["currency"],
			args["amount"],
			args["status"])
	}

	qs = qs + "&appsecret=" + that.Conf.PayKey
	fmt.Println(qs)
	sg := strings.ToLower(helper.GetMD5Hash(helper.GetMD5Hash(helper.GetMD5Hash(qs))))
	return sg
}

//VnQrDetail 根据订单获取扫码支付的页面数据
func VnQrDetail(orderNo string) (qrPayResp, error) {

	merchantNo := meta.Finance["vn"]["merchan_no"].(string) //商户号
	payKey := meta.Finance["vn"]["key"].(string)            //加密密钥
	apiUrl := meta.Finance["vn"]["api"].(string)            //请求的域名
	appID := meta.Finance["vn"]["app_id"].(string)          //公司apiCode
	merchant := meta.Finance["vn"]["merchan"].(string)      //商户号apiCode

	data := qrPayResp{}
	var path string
	fmt.Println("VnQrDetail qrDetail orderId = ", orderNo)

	now := time.Now()
	recs := map[string]string{
		"merchantNo": merchantNo, // 商户编号
		"orderNo":    orderNo,    // 支付返回的三方的订单号
	}

	tp := fmt.Sprintf("%d", now.UnixMilli())
	recs["timestamp"] = tp
	recs["sign"] = sign(recs, payKey)
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

	path = "/v1/api/pay/detail/"
	uri := fmt.Sprintf("%s%s%s/%s/%s", apiUrl, path, appID, merchant, orderNo)

	v, err := httpDoTimeout("p3 pay", body, "POST", uri, header, time.Second*8)
	if err != nil {
		fmt.Println("VnQrDetail uri = ", uri)
		fmt.Println("VnQrDetail httpDoTimeout err = ", err)
		fmt.Println("VnQrDetail body = ", string(v))
		return data, errors.New(helper.PayServerErr)
	}

	fmt.Println("VnQrDetail body = ", string(v))
	if err = helper.JsonUnmarshal(v, &data); err != nil {
		return data, fmt.Errorf("json format err: %s", err.Error())
	}
	return data, nil
}

// sign 组装加签参数
func sign(args map[string]string, payKey string) string {

	qs := fmt.Sprintf(`merchantNo=%s&orderNo=%s&timestamp=%s`,
		args["merchantNo"], args["orderNo"], args["timestamp"])
	qs = qs + "&appsecret=" + payKey
	fmt.Println(qs)
	sg := strings.ToLower(helper.GetMD5Hash(helper.GetMD5Hash(helper.GetMD5Hash(qs))))
	return sg
}
