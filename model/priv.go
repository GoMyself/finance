package model

import (
	"errors"
	"fmt"
)

func PrivCheck(uri, gid string) error {

	privId, err := meta.MerchantRedis.HGet(ctx, "PrivMap", uri).Result()
	if err != nil {
		return err
	}

	id := fmt.Sprintf("GM%s", gid)
	exists := meta.MerchantRedis.HExists(ctx, id, privId).Val()
	if !exists {
		return errors.New("404")
	}

	return nil
}
