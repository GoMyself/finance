package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	g "github.com/doug-martin/goqu/v9"
)

// TunnelData 财务管理-渠道管理-列表 response structure
type TunnelData struct {
	D []Tunnel_t `json:"d"`
	T int64      `json:"t"`
	S uint16     `json:"s"`
}

func TunnelList(all bool, page uint16, pageSize uint16) (TunnelData, error) {

	ex := g.Ex{
		"prefix": meta.Prefix,
	}

	var data TunnelData
	if all {
		query, _, _ := dialect.From("f_channel_type").Select(colTunnel...).Where(ex).Order(g.C("sort").Asc()).ToSQL()
		err := meta.MerchantDB.Select(&data.D, query)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}

		return data, nil
	}

	if page == 1 {
		query, _, _ := dialect.From("f_channel_type").Select(g.COUNT(1)).Where(ex).ToSQL()
		err := meta.MerchantDB.Get(&data.T, query)
		if err != nil && err != sql.ErrNoRows {
			return data, pushLog(err, helper.DBErr)
		}

		if data.T == 0 {
			return data, nil
		}
	}

	offset := (page - 1) * pageSize
	query, _, _ := dialect.From("f_channel_type").Select(colTunnel...).
		Where(ex).Order(g.C("sort").Asc()).Offset(uint(offset)).Limit(uint(pageSize)).ToSQL()
	err := meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	data.S = pageSize
	return data, nil
}

func TunnelUpdate(param map[string]string) error { // 校验渠道id和通道id是否存在

	record := g.Record{
		"sort": param["sort"],
	}

	query, _, _ := dialect.Update("f_channel_type").Set(record).Where(g.Ex{"id": param["id"], "prefix": meta.Prefix}).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	_, err = meta.MerchantRedis.HSet(ctx, "payment_discount", param["id"], param["discount"]).Result()
	if err != nil {
		_ = pushLog(err, helper.RedisErr)
	}

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
