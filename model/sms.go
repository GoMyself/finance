package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"
	"github.com/go-redis/redis/v8"
)

func CheckSmsCaptcha(ip, sid, phone, code string) (bool, error) {

	key := fmt.Sprintf("%s:sms:%s%s%s", meta.Prefix, phone, ip, sid)
	val, err := meta.MerchantRedis.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return false, errors.New(helper.CaptchaErr)
	}

	if code == val {
		return true, nil
	}

	return false, errors.New(helper.CaptchaErr)
}
