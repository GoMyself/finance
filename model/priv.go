package model

import (
	"errors"
	"fmt"
)

func PrivCheck(uri, gid string) error {

	privMapKey := fmt.Sprintf("%s:priv:PrivMap", meta.Prefix)
	privId, err := meta.MerchantRedis.HGet(ctx, privMapKey, uri).Result()
	if err != nil {
		return err
	}

	id := fmt.Sprintf("%s:priv:GM%s", meta.Prefix, gid)
	exists := meta.MerchantRedis.HExists(ctx, id, privId).Val()
	if !exists {
		return errors.New("404")
	}

	return nil
}
