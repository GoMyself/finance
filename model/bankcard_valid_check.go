package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"
	"log"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
)

type bankcard_check_t struct {
	BankCard string `json:"bankCard"`
	Name     string `json:"name"`
	BankCode string `json:"bankCode"`
	Sign     string `json:"sign"`
}

func BankcardCheck(fctx *fasthttp.RequestCtx, bankCard, bankCode, name string) error {

	ts := fmt.Sprintf("%d", fctx.Time().In(loc).UnixMilli())

	data := bankcard_check_t{
		BankCard: bankCard,
		BankCode: bankCode,
		Name:     name,
	}

	id, err := BankcardTaskCreate(ts, data)
	if err != nil {
		return err
	}

	for i := 0; i < 5; i++ {

		ts = fmt.Sprintf("%d", fctx.Time().In(loc).UnixMilli())

		valid, err := BankcardTaskQuery(ts, id)
		if err == nil {
			if valid {
				return errors.New(helper.Success)
			} else {
				return errors.New(helper.Failure)
			}
		}

		time.Sleep(2 * time.Second)
	}

	return nil
}

func BankcardTaskQuery(ts, id string) (bool, error) {

	headers := map[string]string{
		"Timestamp":    ts,
		"Nonce":        helper.GenId(),
		"Content-Type": "application/json",
	}

	str := fmt.Sprintf("orderNo=%s&timestamp=%s&appsecret=%s", id, ts, meta.CardValid.Key)
	uri := fmt.Sprintf("%s/bank/result/query", meta.CardValid.URL)

	sign := helper.GetMD5Hash(helper.GetMD5Hash(helper.GetMD5Hash(str)))

	b := fmt.Sprintf("{\"orderNo\":\"%s\", \"sign\":\"%s\"}", id, sign)

	body, statusCode, err := helper.HttpDoTimeout([]byte(b), "POST", uri, headers, 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	if statusCode != 200 {
		return false, err
	}

	value := fastjson.MustParseBytes(body)
	msg := string(value.GetStringBytes("msg"))
	code := string(value.GetStringBytes("code"))

	if code == "0000" {
		data := value.GetBool("data")
		return data, nil
	}

	return false, errors.New(msg)
}

func BankcardTaskCreate(ts string, res bankcard_check_t) (string, error) {

	headers := map[string]string{
		"Timestamp":    ts,
		"Nonce":        helper.GenId(),
		"Content-Type": "application/json",
	}

	str := fmt.Sprintf("bankCode=%s&bankCard=%s&name=%s&timestamp=%s&appsecret=%s", res.BankCode, res.BankCard, res.Name, ts, meta.CardValid.Key)
	uri := fmt.Sprintf("%s/bank/check/create", meta.CardValid.URL)

	res.Sign = helper.GetMD5Hash(helper.GetMD5Hash(helper.GetMD5Hash(str)))

	b, err := helper.JsonMarshal(res)
	if err != nil {
		log.Fatal(err)
	}

	body, statusCode, err := helper.HttpDoTimeout(b, "POST", uri, headers, 5*time.Second)
	if err != nil {
		return "", err
	}
	if statusCode != 200 {
		return "", err
	}

	value := fastjson.MustParseBytes(body)
	data := string(value.GetStringBytes("data"))
	code := string(value.GetStringBytes("code"))

	if code == "0000" {
		return data, nil
	}

	return "", err
}
