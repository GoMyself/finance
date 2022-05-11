package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"
)

func ChannelTypeCreateCache() {

	var reocrd []Tunnel_t

	query, _, _ := dialect.From("f_channel_type").Select("id", "name", "sort", "promo_state").ToSQL()
	err := meta.MerchantDB.Select(&reocrd, query)
	if err != nil {
		fmt.Println("CreateChannelType meta.MerchantDB.Select = ", err.Error())
		return
	}

	pipe := meta.MerchantRedis.TxPipeline()

	for _, value := range reocrd {

		key := "p:c:t:" + value.ID
		val := map[string]interface{}{
			"promo_state": value.PromoState,
			"sort":        value.Sort,
			"name":        value.Name,
			"id":          value.ID,
		}

		pipe.Unlink(ctx, key)
		pipe.HMSet(ctx, key, val)
		pipe.Persist(ctx, key)
	}
	_, err = pipe.Exec(ctx)
	pipe.Close()

	if err != nil {
		fmt.Println("pipe.Exec = ", err.Error())
	}
}

func ChannelTypeById(id string) (map[string]string, error) {

	key := "p:c:t:" + id
	field := []string{"promo_state", "sort", "name", "id"}
	res := map[string]string{}

	pipe := meta.MerchantRedis.Pipeline()
	ex_temp := pipe.Exists(ctx, key)
	rs_temp := pipe.HMGet(ctx, key, field...)
	_, err := pipe.Exec(ctx)
	pipe.Close()

	if err != nil {
		return res, errors.New(helper.RedisErr)
	}

	if ex_temp.Val() == 0 {
		return res, errors.New(helper.RecordNotExistErr)
	}

	recs := rs_temp.Val()

	for k, v := range field {

		if val, ok := recs[k].(string); ok {
			res[v] = val
		} else {
			res[v] = ""
		}
	}

	return res, nil
}
