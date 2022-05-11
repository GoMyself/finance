package model

import (
	"database/sql"
	"errors"
	"strings"

	"finance/contrib/helper"

	g "github.com/doug-martin/goqu/v9"
	"github.com/go-redis/redis/v8"
)

type PaymentIDChannelID struct {
	PaymentID string `db:"id" json:"id"`
	ChannelId string `db:"channel_id" json:"channel_id"`
}

type ChannelDevice struct {
	ID        string `db:"id" json:"id"`
	PaymentId string `db:"payment_id" json:"payment_id"`
	DeviceId  string `db:"device_id" json:"device_id"`
}

type channelCate struct {
	PaymentID string `db:"id" json:"id"`
	CateID    string `db:"cate_id" json:"cate_id"`
}

func ChannelList(cateID, chanID string, device []string) ([]Payment_t, error) {

	var data []Payment_t

	ex := g.Ex{
		"prefix": meta.Prefix,
	}

	if cateID != "0" {
		ex["cate_id"] = cateID
	}

	if chanID != "0" {
		ex["channel_id"] = chanID
	}

	if len(device) > 0 {
		var ids []string
		exDevice := g.Ex{
			"prefix":    meta.Prefix,
			"device_id": device,
		}

		query, _, _ := dialect.From("f_channel_device").
			Select("payment_id").
			GroupBy("payment_id").
			Where(exDevice).ToSQL()
		err := meta.MerchantDB.Select(&ids, query)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}

		if len(ids) == 0 {
			return data, nil
		}

		ex["id"] = ids
	}

	query, _, _ := dialect.From("f_payment").Select(colPayment...).Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}
	ll := len(data)

	if ll > 0 {

		res := make([]*redis.StringCmd, ll)
		pipe := meta.MerchantRedis.Pipeline()
		for i, v := range data {
			key := "p:c:t:" + v.ChannelID
			res[i] = pipe.HGet(ctx, key, "name")
		}

		pipe.Exec(ctx)
		pipe.Close()

		for i := 0; i < ll; i++ {
			data[i].ChannelName = res[i].Val()
		}
	}

	return data, nil
}

func ChannelInsert(param map[string]string, device []string) error {

	// payment表通cate_id和channel_id的记录只能有一条
	p, err := ChanByCateAndChan(param["cate_id"], param["channel_id"])
	if err != nil {
		return err
	}

	if len(p.ID) != 0 {
		return errors.New(helper.CateIDAndChannelIDErr)
	}

	record := g.Record{
		"id":          param["id"],
		"cate_id":     param["cate_id"],
		"channel_id":  param["channel_id"],
		"quota":       "0",
		"gateway":     "",
		"fmin":        param["fmin"],
		"fmax":        param["fmax"],
		"st":          param["st"],
		"et":          param["et"],
		"created_at":  param["created_at"],
		"state":       "0",
		"amount":      "0",
		"sort":        param["sort"],
		"comment":     param["comment"],
		"amount_list": param["amount_list"],
		"prefix":      meta.Prefix,
	}

	var dr []g.Record
	for _, v := range device {
		dr = append(dr, g.Record{
			"id":         helper.GenId(),
			"payment_id": param["id"],
			"device_id":  v,
			"prefix":     meta.Prefix,
		})
	}

	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.TransErr)
	}

	query, _, _ := dialect.Insert("f_payment").Rows(record).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.TransErr)
	}

	if len(device) > 0 {
		query, _, _ = dialect.Insert("f_channel_device").Rows(dr).ToSQL()
		_, err = tx.Exec(query)
		if err != nil {
			_ = tx.Rollback()
			return pushLog(err, helper.TransErr)
		}
	}

	err = tx.Commit()
	if err != nil {
		return pushLog(err, helper.TransErr)
	}

	// _ = CacheRefreshPayment(param["id"])
	return nil
}

func ChannelUpdate(param map[string]string, device []string) error {

	record := g.Record{
		"fmin":        param["fmin"],
		"fmax":        param["fmax"],
		"st":          param["st"],
		"et":          param["et"],
		"sort":        param["sort"],
		"comment":     param["comment"],
		"devices":     strings.Join(device, ","),
		"amount_list": param["amount_list"],
	}

	var dr []g.Record
	for _, v := range device {
		dr = append(dr, g.Record{
			"id":         helper.GenId(),
			"payment_id": param["id"],
			"device_id":  v,
			"prefix":     meta.Prefix,
		})
	}

	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.TransErr)
	}

	ex := g.Ex{
		"prefix": meta.Prefix,
		"id":     param["id"],
	}

	query, _, _ := dialect.Update("f_payment").Set(record).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.TransErr)
	}

	ex = g.Ex{
		"prefix":     meta.Prefix,
		"payment_id": param["id"],
	}
	query, _, _ = dialect.Delete("f_channel_device").Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.TransErr)
	}

	if len(device) > 0 {
		query, _, _ = dialect.Insert("f_channel_device").Rows(dr).ToSQL()
		_, err = tx.Exec(query)
		if err != nil {
			_ = tx.Rollback()
			return pushLog(err, helper.TransErr)
		}
	}

	err = tx.Commit()
	if err != nil {
		return pushLog(err, helper.TransErr)
	}

	// _ = CacheRefreshPayment(param["id"])
	return nil
}

func ChannelDelete(id string) error {

	channel, err := ChanExistsByID(id)
	if err != nil {
		return err
	}

	if len(channel.ID) == 0 {
		return errors.New(helper.RecordNotExistErr)
	}

	if channel.State == "1" {
		return errors.New(helper.DeleteMustCloseFirst)
	}

	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.TransErr)
	}

	ex := g.Ex{
		"prefix": meta.Prefix,
		"id":     id,
	}
	query, _, _ := dialect.Delete("f_payment").Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.TransErr)
	}

	ex = g.Ex{
		"prefix":     meta.Prefix,
		"payment_id": id,
	}
	query, _, _ = dialect.Delete("f_channel_device").Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.TransErr)
	}

	err = tx.Commit()
	if err != nil {
		return pushLog(err, helper.TransErr)
	}

	// _ = CacheRefreshPayment(id)
	return nil
}

func ChannelSet(id, state string) error {

	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return pushLog(err, helper.TransErr)
	}

	ex := g.Ex{
		"prefix": meta.Prefix,
		"id":     id,
	}
	query, _, _ := dialect.Update("f_payment").Set(g.Record{"state": state}).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return pushLog(err, helper.TransErr)
	}

	if state == "0" {
		ex = g.Ex{
			"prefix":     meta.Prefix,
			"payment_id": id,
		}
		query, _, _ = dialect.Update("f_vip").Set(g.Record{"state": state}).Where(ex).ToSQL()
		_, err = tx.Exec(query)
		if err != nil {
			_ = tx.Rollback()
			return pushLog(err, helper.TransErr)
		}
	}

	err = tx.Commit()
	if err != nil {
		return pushLog(err, helper.TransErr)
	}

	// refresh cache
	var levels []string
	ex = g.Ex{
		"prefix":     meta.Prefix,
		"payment_id": id,
	}
	query, _, _ = dialect.From("f_vip").Select("vip").
		Where(ex).GroupBy("vip").ToSQL()
	err = meta.MerchantDB.Select(&levels, query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	for _, level := range levels {
		Create(level)
	}

	_ = CacheRefreshPayment(id)

	_ = CacheRefreshPaymentBanks(id)
	return nil
}

// ChanByCateAndChan 通过cate id和channel id查找cate
func ChanByCateAndChan(cateId, ChanId string) (Payment_t, error) {

	var channel Payment_t

	query, _, _ := dialect.From("f_payment").Select(colPayment...).
		Where(g.Ex{"cate_id": cateId, "channel_id": ChanId, "prefix": meta.Prefix}).ToSQL()
	err := meta.MerchantDB.Get(&channel, query)
	if err != nil && err != sql.ErrNoRows {
		return channel, pushLog(err, helper.DBErr)
	}

	return channel, nil
}

func ChanByID(id string) (Payment_t, error) {

	var channel Payment_t

	ex := g.Ex{
		"id":     id,
		"prefix": meta.Prefix,
	}

	query, _, _ := dialect.From("f_payment").Select(colPayment...).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&channel, query)
	if err != nil && err != sql.ErrNoRows {
		return channel, pushLog(err, helper.DBErr)
	}

	return channel, nil
}

func ChanExistsByID(id string) (Payment_t, error) {

	var channel Payment_t
	ex := g.Ex{
		"prefix": meta.Prefix,
		"id":     id,
	}
	query, _, _ := dialect.From("f_payment").Select(colPayment...).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&channel, query)
	if err != nil && err != sql.ErrNoRows {
		return channel, pushLog(err, helper.DBErr)
	}

	return channel, nil
}

func ChanWithdrawByCateID(cid string) (Payment_t, error) {

	var channel Payment_t

	ex := g.Ex{
		"cate_id":    cid,
		"channel_id": "7",
		"prefix":     meta.Prefix,
	}
	query, _, _ := dialect.From("f_payment").Select(colPayment...).Where(ex).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&channel, query)
	if err != nil && err != sql.ErrNoRows {
		return channel, pushLog(err, helper.DBErr)
	}

	return channel, nil
}

func ChanByCateID(cid string) (Payment_t, error) {

	var channel Payment_t
	ex := g.Ex{
		"prefix":  meta.Prefix,
		"cate_id": cid,
	}
	query, _, _ := dialect.From("f_payment").Select(colPayment...).Where(ex).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&channel, query)
	if err != nil && err != sql.ErrNoRows {
		return channel, pushLog(err, helper.DBErr)
	}

	return channel, nil
}

func PaymentIDMapToChanID(pids []string) (map[string]string, error) {

	var (
		data []PaymentIDChannelID
		res  = map[string]string{}
	)

	if len(pids) == 0 {
		return res, nil
	}

	// 构造查询用户数量的sql
	query, _, _ := dialect.From("f_payment").
		Select([]interface{}{"id", "channel_id"}...).
		Where(g.Ex{"id": g.Op{"in": pids}, "prefix": meta.Prefix}).GroupBy("id").ToSQL()

	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return nil, pushLog(err, helper.DBErr)
	}

	for _, v := range data {
		if _, ok := res[v.PaymentID]; !ok {
			res[v.PaymentID] = v.ChannelId
		}
	}

	return res, err
}

// 批量获取存款通道的渠道id和name
func channelCateMap(pids []string) (map[string]CateIDAndName, error) {

	var (
		data []channelCate
		pc   = make(map[string]string)
		res  = make(map[string]CateIDAndName)
	)

	if len(pids) == 0 {
		return res, nil
	}

	ex := g.Ex{
		"id":     g.Op{"in": pids},
		"prefix": meta.Prefix,
	}
	query, _, _ := dialect.From("f_payment").Select("id", "cate_id").Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return res, pushLog(err, helper.DBErr)
	}

	if len(data) == 0 {
		return res, nil
	}

	// 先查询pid对应的cate_id
	var cids = make([]string, 0, len(data))
	for _, v := range data {
		if _, ok := pc[v.PaymentID]; !ok {
			pc[v.PaymentID] = v.CateID
		}
		cids = append(cids, v.CateID)
	}

	// 通过cate_id查询cate_name
	c, err := CateIDAndNameByCIDS(cids)
	if err != nil {
		return res, nil
	}

	for k, v := range pc {
		if vv, ok := c[v]; ok {
			res[k] = vv
		}
	}

	return res, err
}
