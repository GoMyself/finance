package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"

	"lukechampine.com/frand"
)

//生成 交易码
func TransacCodeCreate() {

	total := 30000
	vec := map[string]bool{}

	for true {

		code := frand.Intn(899999) + 100000
		key := fmt.Sprintf("%d", code)

		if _, ok := vec[key]; !ok {
			vec[key] = true
		}

		if len(vec) >= total {
			break
		}
	}

	pipe := meta.MerchantRedis.Pipeline()
	pipe.Del(ctx, meta.Prefix+":manual:code")
	for code, _ := range vec {
		pipe.LPush(ctx, meta.Prefix+":manual:code", code)
	}
	_, err := pipe.Exec(ctx)
	pipe.Close()
	if err != nil {
		fmt.Println("CreateCode pipe.Exec = ", err.Error())
	}
}

func TransacCodeGet() (string, error) {

	code, err := meta.MerchantRedis.RPopLPush(ctx, meta.Prefix+":manual:code", "manual:code").Result()
	if err != nil {
		return "", errors.New(helper.RecordNotExistErr)
	}

	return code, nil
}
