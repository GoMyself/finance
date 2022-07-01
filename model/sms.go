package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
	"github.com/go-redis/redis/v8"
	"strconv"
	"time"
)

func CheckSmsCaptcha(ip, ts, sid, phone, code string) (bool, error) {

	key := fmt.Sprintf("%s:sms:%s%s%s", meta.Prefix, phone, ip, sid)
	cmd := meta.MerchantRedis.Get(ctx, key)
	val, err := cmd.Result()
	if err != nil && err != redis.Nil {
		return false, pushLog(fmt.Errorf("CheckSmsCaptcha cmd : %s ,error : %s ", cmd.String(), err.Error()), helper.RedisErr)
	}

	if code == val {
		its, _ := strconv.ParseInt(ts, 10, 64)
		tdInsert("sms_log", g.Record{
			"ts":         its,
			"state":      "2",
			"updated_at": time.Now().Unix(),
		})
		return true, nil
	}

	return false, errors.New(helper.CaptchaErr)
}
