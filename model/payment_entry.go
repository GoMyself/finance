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
	"lukechampine.com/frand"

	"github.com/valyala/fasthttp"
)

// 定义全局 payment

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
	)

	WPay.New()     // wPay
	QuickPay.New() // quickPay
	FyPay.New()    // 凤扬支付
	UzPay.New()    // uzPay
	USDTPay.New()  // USDT1
	YFB.New()      // yfb 支付
	YNPAY.New()

	paymentRoute = map[string]Payment{
		"1":  UzPay,
		"6":  WPay,
		"9":  FyPay,
		"10": QuickPay,
		"11": USDTPay,
		"7":  YFB,
		"16": YNPAY,
	}
}

//Pay 发起支付公共入口
func Pay(pLog *paymentTDLog, user Member, p FPay, amount, bid string) (paymentDepositResp, error) {

	data := paymentDepositResp{}

	payment, ok := paymentRoute[p.CateID]
	if !ok {
		return data, errors.New(helper.NoPayChannel)
	}

	ch := paymentChannelMatch(p.ChannelID)
	pLog.Merchant = payment.Name()
	pLog.Channel = string(ch)

	// 检查存款金额是否符合范围
	a, ok := validator.CheckFloatScope(amount, p.Fmin, p.Fmax)
	if !ok {
		return data, errors.New(helper.AmountOutRange)
	}

	amount = a.String()

	// online, remit, unionPay 需要判断是否传银行卡信息
	switch ch {
	case online, remit, unionpay:
		if bid == "0" || bid == "" {
			return data, errors.New(helper.BankNameOrCodeErr)
		}
	default:
		bid = ""
	}

	// 生成我方存款订单号
	pLog.OrderID = helper.GenId()

	// 检查用户的存款行为是否过于频繁
	err := cacheDepositProcessing(user.UID, time.Now().Unix())
	if err != nil {
		return data, err
	}
	// 向渠道方发送存款订单请求
	data, err = payment.Pay(pLog, ch, amount, bid)
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

func httpDoTimeout(requestBody []byte, method string, requestURI string, headers map[string]string, timeout time.Duration, log *paymentTDLog) ([]byte, error) {

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()

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

	log.RequestURL = requestURI
	log.RequestBody = string(requestBody)
	log.ResponseBody = string(respBody)
	log.ResponseCode = code

	if err != nil {
		return nil, fmt.Errorf("send http request error: [%v]", err)
	}

	if code != fasthttp.StatusOK {
		return nil, fmt.Errorf("bad http response code: [%d]", code)
	}

	return respBody, nil
}

// 写入日志
func paymentPushLog(data *paymentTDLog) {

	if data.Error == "" {
		data.Level = "info"
	} else {
		data.Level = "error"
	}

	logInfo := map[string]string{
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
	}

	//fmt.Printf("%v \n", logInfo)
	//err := tdlog.WriteLog(paymentLogTag, logInfo)
	//if err != nil {
	//	fmt.Println("logging payment error: ", err.Error())
	//}

	_ = meta.Zlog.Post(esPrefixIndex(paymentLogTag), logInfo)
}

func paymentChannelMatch(cid string) paymentChannel {

	if v, ok := channels[cid]; ok {
		return v
	}

	return ""
}

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
func BankCards(channelBankID string) (BankCard, error) {

	card := BankCard{}
	ex := g.Ex{
		"channel_bank_id": channelBankID,
		"state":           1,
		"prefix":          meta.Prefix,
	}
	query, _, _ := dialect.From("f_bankcards").Select("id", "card_no", "bank_addr", "real_name", "name", "max_amount").
		Limit(1).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&card, query)
	if err != nil {
		return BankCard{}, err
	}

	return card, nil
}

// DepositManualRemark 生成不重复的附言码 三天内不重复
func DepositManualRemark(cardID string) (string, error) {

	code := 0
	for true {

		code = frand.Intn(899999) + 100000
		key := ManualRemarkCodeKey(cardID, code)

		rs, err := meta.MerchantRedis.Exists(ctx, key).Result()
		if err != nil {
			return "", err
		}

		if rs == 1 {
			continue
		}

		meta.MerchantRedis.Set(ctx, key, 1, 72*time.Hour)
		break
	}

	return strconv.Itoa(code), nil
}

func ManualRemarkCodeKey(bankcardID string, code int) string {
	return fmt.Sprintf("MR:%s:%d", bankcardID, code)
}

func PushWithdrawSuccess(uid string, amount float64) error {
	msg := fmt.Sprintf(`{"amount": %.4f, "flags":"withdraw"}`, amount)
	err := meta.Nats.Publish(uid, []byte(msg))
	_ = meta.Nats.Flush()
	fmt.Printf("Nats send[%s] a message: %s, error: %v\n", uid, msg, err)
	return err
}

func PushDepositSuccess(uid string, amount float64) error {
	msg := fmt.Sprintf(`{"amount": %.4f, "flags":"deposit"}`, amount)
	err := meta.Nats.Publish(uid, []byte(msg))
	_ = meta.Nats.Flush()
	fmt.Printf("Nats send[%s] a message: %s, error: %v\n", uid, msg, err)
	return err
}
