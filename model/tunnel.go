package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"fmt"

	g "github.com/doug-martin/goqu/v9"
)

// TunnelData 财务管理-渠道管理-列表 response structure

func TunnelList() ([]Tunnel_t, error) {

	var data []Tunnel_t

	ex := g.Ex{
		"prefix": meta.Prefix,
	}

	query, _, _ := dialect.From("f_channel_type").Select(colTunnel...).Where(ex).Order(g.L("sort").Asc()).ToSQL()
	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}

func TunnelUpdate(id, state, discount, seq string) error { // 校验渠道id和通道id是否存在

	record := g.Record{}

	if state != "" {
		record["promo_state"] = state
	}
	if discount != "" {
		record["promo_discount"] = discount
	}
	if seq != "" {
		record["sort"] = seq
	}

	query, _, _ := dialect.Update("f_channel_type").Set(record).Where(g.Ex{"id": id, "prefix": meta.Prefix}).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	TunnelUpdateCache()
	for i := 1; i < 11; i++ {
		level := fmt.Sprintf("%d", i)
		Create(level)
	}
	ChannelTypeCreateCache()
	return nil
}

func TunnelUpdateCache() error {

	res, err := TunnelList()
	if err != nil {
		return err
	}

	key := meta.Prefix + ":tunnel:All"
	val, _ := helper.JsonMarshal(res)

	pipe := meta.MerchantRedis.Pipeline()
	pipe.Unlink(ctx, key)
	pipe.Set(ctx, key, string(val), 0)
	pipe.Exec(ctx)
	pipe.Close()

	return nil
}

// 获取三方通道
func TunnelByID(id string) (Tunnel_t, error) {

	tunnel := Tunnel_t{}
	query, _, _ := dialect.From("f_channel_type").Select(colTunnel...).Where(g.Ex{"id": id, "prefix": meta.Prefix}).ToSQL()
	err := meta.MerchantDB.Get(&tunnel, query)
	if err != nil && err != sql.ErrNoRows {
		return tunnel, pushLog(err, helper.DBErr)
	}

	return tunnel, nil
}

func TunnelAndChannelGetName(cateID, channelID string) (string, string, error) {
	// 三方渠道
	cate, err := CateByID(cateID)
	if err != nil {
		return "", "", err
	}

	if len(cate.ID) == 0 {
		return "", "", errors.New(helper.RecordNotExistErr)
	}

	// 三方通道
	channel, err := TunnelByID(channelID)
	if err != nil {
		return "", "", err
	}

	if len(channel.ID) == 0 {
		return "", "", errors.New(helper.RecordNotExistErr)
	}

	return cate.Name, channel.Name, nil
}
