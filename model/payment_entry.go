package model

import (
	"errors"
	"finance/contrib/helper"
	"finance/contrib/validator"
	"fmt"
	"strconv"
	"time"

	g "github.com/doug-martin/goqu/v9"
	"github.com/shopspring/decimal"

	"github.com/valyala/fasthttp"
)

// Payment 接口
type Payment interface {
	// Name 支付通道名称
	Name() string
	// New 初始化 通道配置
	New()
	// Pay 发起支付
	Pay(orderId, paymentChannel, amount, bid string) (paymentDepositResp, error)
	// Withdraw 发起代付
	Withdraw(param WithdrawAutoParam) (paymentWithdrawalRsp, error)
	// PayCallBack 支付回调
	PayCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error)
	// WithdrawCallBack 代付回调
	WithdrawCallBack(fctx *fasthttp.RequestCtx) (paymentCallbackResp, error)
}

//New 初始化配置
func NewPayment() {

	var (
		WPay     = new(WPayment)
		QuickPay = new(QuickPayment)
		FyPay    = new(FyPayment)
		UzPay    = new(UzPayment)
		USDTPay  = new(USDTPayment)
		YFB      = new(YfbPayment)
		YNPAY    = new(YNPayment)
		VtPAY    = new(VtPayment)
		JybPAY   = new(JybPayment)
		VnPAY    = new(VnPayment)
		DbPay    = new(DbPayment)
	)

	WPay.New()     // wPay
	QuickPay.New() // quickPay
	FyPay.New()    // 凤扬支付
	UzPay.New()    // uzPay
	USDTPay.New()  // USDT1
	YFB.New()      // yfb 支付
	YNPAY.New()
	VtPAY.New()  // vtech支付
	JybPAY.New() // 918支付
	VnPAY.New()  // 越南支付
	DbPay.New()  // 帝宝支付

	paymentRoute = map[string]Payment{
		"1":  UzPay,
		"6":  WPay,
		"9":  FyPay,
		"10": QuickPay,
		"11": USDTPay,
		"7":  YFB,
		"16": YNPAY,
		"17": VtPAY,
		"18": JybPAY,
		"19": VnPAY,
		"20": DbPay,
	}
}

//Pay 发起支付公共入口
func Pay(user Member, p FPay, amount, bid string) (paymentDepositResp, error) {

	data := paymentDepositResp{}

	payment, ok := paymentRoute[p.CateID]
	if !ok {
		return data, errors.New(helper.NoPayChannel)
	}

	fmt.Println("Pay payment = ", payment)
	fmt.Println("Pay p = ", p)

	ch, err := ChannelTypeById(p.ChannelID)
	if err != nil {
		return data, errors.New(helper.ChannelNotExist)
	}

	fmt.Println("Pay ch = ", ch)

	// 检查存款金额是否符合范围
	a, ok := validator.CheckFloatScope(amount, p.Fmin, p.Fmax)
	if !ok {
		return data, errors.New(helper.AmountOutRange)
	}

	amount = a.String()

	// online, remit, unionPay 需要判断是否传银行卡信息
	switch p.ChannelID {
	case "3", "5", "4", "8":
		if bid == "0" || bid == "" {
			return data, errors.New(helper.BankNameOrCodeErr)
		}
	default:
		bid = ""
	}

	// 生成我方存款订单号
	orderId := helper.GenId()

	fmt.Println("Pay orderId = ", orderId)

	/*
		// 检查用户的存款行为是否过于频繁
		err = cacheDepositProcessing(user.UID, time.Now().Unix())
		if err != nil {
			return data, err
		}
	*/
	// 向渠道方发送存款订单请求
	data, err = payment.Pay(orderId, p.ChannelID, amount, bid)
	fmt.Println("Pay  payment.Pay err = ", err)
	if err != nil {
		return data, err
	}

	return data, nil
}

//WithdrawGetPayment 提款获取通道 cateID
func WithdrawGetPayment(cateID string) (Payment, error) {
	p, ok := paymentRoute[cateID]
	if ok {
		return p, nil
	}
	return p, errors.New(helper.CateNotExist)
}

func httpDoTimeout(merchant string, requestBody []byte, method string, requestURI string, headers map[string]string, timeout time.Duration) ([]byte, error) {

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	fmt.Println("****")
	fmt.Println("requestURI = ", requestURI)
	fmt.Println("requestBody = ", string(requestBody))
	defer func() {
		fasthttp.ReleaseResponse(resp)
		fasthttp.ReleaseRequest(req)
	}()

	req.SetRequestURI(requestURI)
	req.Header.SetMethod(method)

	switch method {
	case "POST":
		req.SetBody(requestBody)
	}

	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	// time.Second * 30
	err := fc.DoTimeout(req, resp, timeout)

	code := resp.StatusCode()
	respBody := resp.Body()

	pLog := paymentTDLog{
		Merchant:   merchant,
		Flag:       "2",
		Lable:      paymentLogTag,
		RequestURL: requestURI,
	}
	// 记录请求日志
	defer func() {

		pLog.ResponseBody = string(respBody)
		pLog.ResponseCode = code
		paymentPushLog(pLog)
	}()
	fmt.Println("body = ", string(respBody))
	if err != nil {
		return respBody, fmt.Errorf("send http request error: [%v]", err)
	}

	if code != fasthttp.StatusOK {
		return respBody, fmt.Errorf("bad http response code: [%d]", code)
	}

	return respBody, nil
}

// 写入日志
func paymentPushLog(data paymentTDLog) {

	ts := time.Now()

	if data.Error == "" {
		data.Level = "info"
	} else {
		data.Level = "error"
	}

	fields := g.Record{
		"username":      data.Username,
		"lable":         paymentLogTag,
		"order_id":      data.OrderID,
		"level":         data.Level,
		"error":         data.Error,
		"response_body": data.ResponseBody,
		"response_code": strconv.Itoa(data.ResponseCode),
		"request_body":  data.RequestBody,
		"request_url":   data.RequestURL,
		"merchant":      data.Merchant,
		"channel":       data.Channel,
		"flag":          data.Flag,
		"ts":            ts.In(loc).UnixMicro(),
	}

	//fmt.Printf("%v \n", logInfo)
	//err := tdlog.WriteLog(paymentLogTag, logInfo)
	//if err != nil {
	//	fmt.Println("logging payment error: ", err.Error())
	//}

	query, _, _ := dialect.Insert("finance_log").Rows(&fields).ToSQL()
	fmt.Println(query)
	_, err1 := meta.MerchantTD.Exec(query)
	if err1 != nil {
		fmt.Println("insert finance_log = ", err1.Error())
	}

	//_ = meta.Zlog.Post(esPrefixIndex(paymentLogTag), logInfo)
}

/*
func paymentChannelMatch(cid string) string {

	if v, ok := channels[cid]; ok {
		return v
	}

	return ""
}
*/

// 金额对比
func compareAmount(compare, compared string, cent int64) error {

	ca, err := decimal.NewFromString(compare)
	if err != nil {
		return errors.New("parse amount error")
	}

	ra, err := decimal.NewFromString(compared)
	if err != nil {
		return errors.New("parse amount error")
	}

	// 数据库的金额是k为单位 ra.Mul(decimal.NewFromInt(1000))
	if ca.Cmp(ra.Mul(decimal.NewFromInt(cent))) != 0 {
		return errors.New("invalid amount")
	}

	return nil
}

func valid(p map[string]string, key []string) bool {

	for _, v := range key {
		_, ok := p[v]
		if !ok {
			return false
		}
	}

	return true
}

// BankCards 获取返回前台的银行卡
func BankCards(channelBankID string) (Bankcard_t, error) {

	card := Bankcard_t{}
	ex := g.Ex{
		"channel_bank_id": channelBankID,
		"state":           1,
		"prefix":          meta.Prefix,
	}
	query, _, _ := dialect.From("f_bankcards").Select("id", "card_no", "bank_addr", "real_name", "name", "max_amount").
		Limit(1).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&card, query)
	if err != nil {
		return card, err
	}

	return card, nil
}

// DepositManualRemark 生成不重复的附言码 三天内不重复
func DepositManualRemark(cardID string) (string, error) {

	code := 0

	return strconv.Itoa(code), nil
}

func PushWithdrawSuccess(uid string, amount float64) error {
	msg := fmt.Sprintf(`{"amount": %.4f, "flags":"withdraw"}`, amount)

	topic := fmt.Sprintf("%s/%s/finance", meta.Prefix, uid)
	err := Publish(topic, []byte(msg))
	if err != nil {
		fmt.Println("merchantNats.Publish finance = ", err.Error())
		return err
	}

	return nil
	/*
		err := meta.Nats.Publish(uid, []byte(msg))
		_ = meta.Nats.Flush()
		fmt.Printf("Nats send[%s] a message: %s, error: %v\n", uid, msg, err)
		return err
	*/
}

func PushDepositSuccess(uid string, amount float64) error {
	msg := fmt.Sprintf(`{"amount": %.4f, "flags":"deposit"}`, amount)

	topic := fmt.Sprintf("%s/%s/finance", meta.Prefix, uid)
	err := Publish(topic, []byte(msg))
	if err != nil {
		fmt.Println("merchantNats.Publish finance = ", err.Error())
		return err
	}

	return nil

	/*
		err := meta.Nats.Publish(uid, []byte(msg))
		_ = meta.Nats.Flush()
		fmt.Printf("Nats send[%s] a message: %s, error: %v\n", uid, msg, err)
		return err
	*/
}
