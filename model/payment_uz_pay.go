package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"
	"sort"
	"time"

	"github.com/valyala/fasthttp"
)

//UzPayment 渠道
type UzPayment struct {
	Conf uzConf
}

type uzConf struct {
	AppID          string
	Name           string
	Domain         string
	Key            string
	PayNotify      string
	PayReturn      string
	WithdrawNotify string
	Channel        map[string]string
}

type uzPayResp struct {
	Success bool   `json:"success"`
	Info    uzInfo `json:"info"`
}

type uzInfo struct {
	Action string `json:"action"`
	QrURL  string `json:"qrurl"`
}

type uzPayCallBack struct {
	OrderID     string `json:"orderid"` // 订单号
	Amount      string `json:"amount"`  // 订单金额
	Service     string `json:"service"` // 充值(collection) or 提现(withdraw)
	Status      string `json:"status"`  // verified = 已完成 & revoked = 被撒销 timeout = 逾时 & processing = 處理中
	Sign        string `json:"sign"`    // 签名
	CreatedTime string `json:"created_time"`
	Oid         string `json:"oid"`
	UserID      string `json:"userid"`
}

//New 初始化配置
func (that *UzPayment) New() {

	//appID := "55834"
	//key := "4db72bcc4dc5a447051c9a9914d22475"
	//
	//if meta.IsDev { // 测试账号
	//	appID = "55845"
	//	key = "52d891adc939fbc9e99ed37da4b7dbcf"
	//}

	appID := meta.Finance["uz"]["app_id"].(string)
	key := meta.Finance["uz"]["key"].(string)

	that.Conf = uzConf{
		AppID:          appID,
		Key:            key,
		Name:           "UZ",
		Domain:         "https://www.uz-pay.com",
		PayNotify:      "%s/finance/callback/uzd",
		WithdrawNotify: "%s/finance/callback/uzw",
		Channel: map[string]string{
			"momo":   "momo",   // momo
			"zalo":   "zalo",   // zalo
			"online": "online", // online
			"remit":  "remit",  // unionpay
		},
	}
}

//Name 渠道名称
func (that *UzPayment) Name() string {
	return that.Conf.Name
}

//Pay 发起支付
func (that *UzPayment) Pay(log *paymentTDLog, ch string, amount, bid string) (paymentDepositResp, error) {

	data := paymentDepositResp{}

	cno, ok := that.Conf.Channel[ch]
	if !ok {
		return data, errors.New(helper.ChannelNotExist)
	}

	if bid == "" {
		bid = "qr"
	}

	params := map[string]string{
		"uid":           that.Conf.AppID,
		"userid":        "125423",
		"amount":        fmt.Sprintf("%s000", amount),
		"orderid":       log.OrderID, //贵司订单编号
		"cate":          string(cno),
		"userip":        "203.208.43.98",
		"from_bankflag": bid,
		"notify":        fmt.Sprintf(that.Conf.PayNotify, meta.Fcallback), // 自定义
	}

	params["sign"] = that.sign(params)
	params["language"] = "vi"

	body, err := helper.JsonMarshal(params)
	if err != nil {
		return data, errors.New(helper.FormatErr)
	}

	header := map[string]string{
		"Content-Type": "application/json",
	}
	query := fmt.Sprintf("%s/Api/collection", that.Conf.Domain)

	res, err := httpDoTimeout(body, "POST", query, header, time.Second*8, log)
	if err != nil {
		return data, err
	}

	var rp uzPayResp
	if err := helper.JsonUnmarshal(res, &rp); err != nil {
		return data, fmt.Errorf("json format err: %s", err.Error())
	}

	if !rp.Success {
		return data, fmt.Errorf("an 3rd-party error occurred")
	}

	if rp.Info.Action != "jump" {
		return data, fmt.Errorf("response action err: %s", rp.Info.Action)
	}

	data.Addr = rp.Info.QrURL
	data.OrderID = log.OrderID

	return data, nil
}

//Withdraw 发起提现
func (that *UzPayment) Withdraw(log *paymentTDLog, arg WithdrawAutoParam) (paymentWithdrawalRsp, error) {

	data := paymentWithdrawalRsp{}

	params := map[string]string{
		"uid":           that.Conf.AppID,         // 接入网站商户专属ID
		"userid":        "125423",                //
		"amount":        arg.Amount,              // 金额，单位 元 ，最多可带两位小数点，如: 1.23, 2.6
		"orderid":       arg.OrderID,             // 接入网站的订单号
		"to_bankflag":   arg.BankCode,            // 请依照「服务支持银行」名称回传
		"to_cardnumber": arg.CardNumber,          // 银行卡号
		"to_cardname":   arg.CardName,            // 持卡人姓名
		"to_province":   "thành phố Hồ Chí Minh", // 请依照「省份和城市定义」省份名称回传
		"to_city":       "thành phố Hồ Chí Minh", // 请依照「省份和城市定义」省份名称回传
		"notify":        fmt.Sprintf(that.Conf.WithdrawNotify, meta.Fcallback),
	}

	params["sign"] = that.sign(params)

	body, err := helper.JsonMarshal(params)
	if err != nil {
		return data, errors.New(helper.FormatErr)
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	uri := fmt.Sprintf("%s/Api/withdraw", that.Conf.Domain)

	v, err := httpDoTimeout(body, "POST", uri, headers, time.Second*8, log)
	if err != nil {
		return data, err
	}

	var res uzPayResp
	err = helper.JsonUnmarshal(v, &res)
	if err != nil {
		return data, err
	}

	if !res.Success {
		return data, fmt.Errorf("an 3rd-party error occurred")
	}

	data.OrderID = arg.OrderID

	return data, nil
}

//PayCallBack 支付回调
func (that *UzPayment) PayCallBack(ctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	data := paymentCallbackResp{
		State: DepositConfirming,
		Sign:  string(ctx.PostArgs().Peek("sign")),
	}

	param := uzPayCallBack{}
	if err := helper.JsonUnmarshal(ctx.PostBody(), &param); err != nil {
		return data, fmt.Errorf("param format err: %s", err.Error())
	}

	switch param.Status {
	case "verified":
		data.State = DepositSuccess
	case "revoked", "timeout":
		data.State = DepositCancelled
	default:
		return data, fmt.Errorf("unknown status: [%s]", param.Status)
	}

	// check signature
	args := map[string]string{
		"amount":       param.Amount,
		"created_time": param.CreatedTime,
		"oid":          param.Oid,
		"orderid":      param.OrderID, //贵司订单编号
		"service":      param.Service,
		"status":       param.Status,
		"userid":       param.UserID,
	}

	if param.CreatedTime == "" {
		delete(args, "created_time")
	}

	if that.sign(args) != param.Sign {
		return data, fmt.Errorf("invalid sign")
	}

	data.OrderID = param.OrderID
	data.Amount = param.Amount

	return data, nil
}

//WithdrawCallBack 提现回调
func (that *UzPayment) WithdrawCallBack(ctx *fasthttp.RequestCtx) (paymentCallbackResp, error) {

	data := paymentCallbackResp{
		State: WithdrawDealing,
		Sign:  string(ctx.PostArgs().Peek("sign")),
	}

	param := uzPayCallBack{}
	if err := helper.JsonUnmarshal(ctx.PostBody(), &param); err != nil {
		return data, fmt.Errorf("param format err: %s", err.Error())
	}

	switch param.Status {
	case "verified":
		data.State = WithdrawSuccess
	case "revoked", "timeout":
		data.State = WithdrawAutoPayFailed
	default:
		return data, fmt.Errorf("unknown status: [%s]", param.Status)
	}

	// check signature
	args := map[string]string{
		"amount":       param.Amount,
		"created_time": param.CreatedTime,
		"oid":          param.Oid,
		"orderid":      param.OrderID, //贵司订单编号
		"service":      param.Service,
		"status":       param.Status,
		"userid":       param.UserID,
	}

	if param.CreatedTime == "" {
		delete(args, "created_time")
	}

	if that.sign(args) != param.Sign {
		return data, fmt.Errorf("invalid sign")
	}

	data.OrderID = param.OrderID
	data.Amount = param.Amount

	return data, nil
}

func (that *UzPayment) sign(p map[string]string) string {

	i := 0
	qs := ""
	keys := make([]string, len(p))

	for k := range p {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, v := range keys {
		qs += fmt.Sprintf("%s=%s&", v, p[v])
	}
	qs += "key=" + that.Conf.Key

	return helper.GetMD5Hash(qs)

}
