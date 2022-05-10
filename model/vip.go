package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"finance/contrib/validator"

	g "github.com/doug-martin/goqu/v9"
	"github.com/go-redis/redis/v8"
)

type Vip_t struct {
	ID          string `db:"id" json:"id"`
	CateID      string `db:"cate_id" json:"cate_id"`
	CateName    string `json:"cate_name"`
	ChannelID   string `db:"channel_id" json:"channel_id"`
	ChannelName string `json:"channel_name"`
	PaymentID   string `db:"payment_id" json:"payment_id"`
	Vip         string `db:"vip" json:"vip"`
	Fmin        string `db:"fmin" json:"fmin"`
	Fmax        string `db:"fmax" json:"fmax"`
	Flags       string `db:"flags" json:"flags"`
	Comment     string `db:"comment" json:"comment"`
	State       string `db:"state" json:"state"`
}

// VipData 财务管理-会员等级通道-列表 response structure

func VipList(level, flags string) ([]Vip_t, error) {

	var data []Vip_t

	ex := g.Ex{
		"vip":    level,
		"flags":  flags,
		"prefix": meta.Prefix,
	}

	query, _, _ := dialect.From("f_vip").Select(colVip...).Where(ex).ToSQL()
	//fmt.Println("query = ", query)
	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	ll := len(data)
	if ll > 0 {

		if flags == "1" {
			res := make([]*redis.StringCmd, ll)
			pipe := meta.MerchantRedis.Pipeline()
			for i, v := range data {
				key := "p:c:t:" + v.ChannelID
				res[i] = pipe.HGet(ctx, key, "name")
			}

			pipe.Exec(ctx)
			pipe.Close()

			for i := 0; i < ll; i++ {

				cateId := data[i].CateID
				data[i].ChannelName = res[i].Val()
				data[i].CateName = meta.MerchantInfo[cateId]
			}
		} else {
			for i := 0; i < ll; i++ {

				cateId := data[i].CateID
				data[i].CateName = meta.MerchantInfo[cateId]
			}
		}

	}

	return data, nil
}

func VipInsert(param map[string]string) error {

	// check cate id and channel id
	channel, err := ChanByCateAndChan(param["cate_id"], param["channel_id"])
	if err != nil {
		return err
	}

	if len(channel.ID) == 0 {
		return errors.New(helper.CateIDAndChannelIDErr)
	}

	/*
		fmt.Println("param[fmin] = ", param["fmin"])
		fmt.Println("channel.Fmin = ", channel.Fmin)
		fmt.Println("channel.Fmax = ", channel.Fmax)
	*/
	// check fmin scope
	fmin, ok := validator.CheckFloatScope(param["fmin"], channel.Fmin, channel.Fmax)
	if !ok {
		return errors.New(helper.AmountOutRange)
	}
	// check fmax scope
	fmax, ok := validator.CheckFloatScope(param["fmax"], channel.Fmin, channel.Fmax)
	if !ok {
		return errors.New(helper.AmountOutRange)
	}

	if !fmin.LessThanOrEqual(fmax) {
		return errors.New(helper.AmountErr)
	}

	var id string
	// 一個level對同一個payment id只能有一條記錄
	ex := g.Ex{"vip": param["vip"], "payment_id": channel.ID}
	query, _, _ := dialect.From("f_vip").Select("id").Where(ex).ToSQL()

	//fmt.Println("query = ", query)
	err = meta.MerchantDB.Get(&id, query)
	if err != nil && err != sql.ErrNoRows {
		return pushLog(err, helper.DBErr)
	}

	if id != "" {
		return errors.New(helper.ChannelExist)
	}

	// 目前只有channel id为7的时候 flag才为2（即代付通道）
	flag := "1"
	if channel.ChannelID == "7" {
		flag = "2"
	}

	record := g.Record{
		"id":         param["id"],
		"cate_id":    param["cate_id"],
		"payment_id": channel.ID,
		"channel_id": param["channel_id"],
		"vip":        param["vip"],
		"fmin":       param["fmin"],
		"fmax":       param["fmax"],
		"comment":    "0",
		"state":      "0",
		"flags":      flag,
		"prefix":     meta.Prefix,
	}
	query, _, _ = dialect.Insert("f_vip").Rows(record).ToSQL()
	_, err = meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return nil
}

func VipUpdate(paymentID string, param map[string]string) error {

	// check cate id and channel id
	channel, err := ChanByID(paymentID)
	if err != nil {
		return err
	}

	if len(channel.ID) == 0 {
		return errors.New(helper.CateIDAndChannelIDErr)
	}

	// check fmin scope
	fmin, ok := validator.CheckFloatScope(param["fmin"], channel.Fmin, channel.Fmax)
	if !ok {
		return errors.New(helper.AmountOutRange)
	}

	// check fmax scope
	fmax, ok := validator.CheckFloatScope(param["fmax"], channel.Fmin, channel.Fmax)
	if !ok {
		return errors.New(helper.AmountOutRange)
	}

	if !fmin.LessThanOrEqual(fmax) {
		return errors.New(helper.AmountErr)
	}

	record := g.Record{
		"fmin": param["fmin"],
		"fmax": param["fmax"],
	}
	query, _, _ := dialect.Update("f_vip").Set(record).Where(g.Ex{"id": param["id"]}).ToSQL()
	_, err = meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return nil
}

func VipDelete(id string) error {

	query, _, _ := dialect.Delete("f_vip").Where(g.Ex{"id": id}).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return nil
}

func VipSet(id, state string, v Vip_t) error {

	// 上级通道关闭的时候不能开启
	if state == "1" {
		channel, err := ChanByID(v.PaymentID)
		if err != nil {
			return err
		}

		if len(channel.ID) == 0 {
			return errors.New(helper.ChannelNotExist)
		}

		if channel.State == "0" {
			return errors.New(helper.ParentChannelClosed)
		}

		// check fmin scope
		fmin, ok := validator.CheckFloatScope(v.Fmin, channel.Fmin, channel.Fmax)
		if !ok {
			return errors.New(helper.AmountOutRange)
		}

		// check fmax scope
		fmax, ok := validator.CheckFloatScope(v.Fmax, channel.Fmin, channel.Fmax)
		if !ok {
			return errors.New(helper.AmountOutRange)
		}

		if !fmin.LessThanOrEqual(fmax) {
			return errors.New(helper.AmountErr)
		}

	}

	query, _, _ := dialect.Update("f_vip").Set(g.Record{"state": state}).Where(g.Ex{"id": id}).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	if v.Flags == "2" {
		// 刷新代付(取款)缓存
		CreateAutomatic(v.Vip)
	} else {
		// 刷新支付(存款)缓存
		Create(v.Vip)
	}

	return nil
}

func VipByID(id string) (Vip_t, error) {

	vip := Vip_t{}

	query, _, _ := dialect.From("f_vip").Select(colVip...).Where(g.Ex{"id": id}).ToSQL()
	err := meta.MerchantDB.Get(&vip, query)
	if err != nil && err != sql.ErrNoRows {
		return vip, pushLog(err, helper.DBErr)
	}

	return vip, nil
}
