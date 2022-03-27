package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"finance/contrib/validator"
	g "github.com/doug-martin/goqu/v9"
)

type Vip struct {
	ID        string `db:"id" json:"id"`
	CateID    string `db:"cate_id" json:"cate_id"`
	CateName  string `db:"-" json:"cate_name"`
	ChannelID string `db:"-" json:"channel_id"`
	PaymentID string `db:"payment_id" json:"payment_id"`
	Vip       string `db:"vip" json:"vip"`
	Fmin      string `db:"fmin" json:"fmin"`
	Fmax      string `db:"fmax" json:"fmax"`
	Flags     string `db:"flags" json:"flags"`
	Comment   string `db:"comment" json:"comment"`
	State     string `db:"state" json:"state"`
}

// VipData 财务管理-会员等级通道-列表 response structure
type VipData struct {
	D []Vip  `json:"d"`
	T int64  `json:"t"`
	S uint16 `json:"s"`
}

func VipList(vip, chanName string, page, pageSize uint16, chanIds []string) (VipData, error) {

	var data VipData

	ex := g.Ex{
		"vip":    vip,
		"prefix": meta.Prefix,
	}

	if chanName != "" {
		var cateID string
		exFC := g.Ex{
			"name":   chanName,
			"prefix": meta.Prefix,
		}
		query, _, _ := dialect.From("f_category").Select("id").Where(exFC).ToSQL()
		err := meta.MerchantDB.Get(&cateID, query)
		if err != nil && err != sql.ErrNoRows {
			return data, pushLog(err, helper.DBErr)
		}

		if cateID == "" {
			return data, nil
		}

		ex["cate_id"] = cateID
	}

	if len(chanIds) > 0 {
		var pids []string
		exCI := g.Ex{
			"prefix":     meta.Prefix,
			"channel_id": chanIds,
		}
		query, _, _ := dialect.From("f_payment").Select("id").Where(exCI).GroupBy("id").ToSQL()
		err := meta.MerchantDB.Select(&pids, query)
		if err != nil && err != sql.ErrNoRows {
			return data, pushLog(err, helper.DBErr)
		}

		if len(pids) == 0 {
			return data, nil
		}

		ex["payment_id"] = pids
	}

	if page == 1 {
		query, _, _ := dialect.From("f_vip").Select(g.COUNT(1)).Where(ex).ToSQL()
		err := meta.MerchantDB.Get(&data.T, query)
		if err != nil && err != sql.ErrNoRows {
			return data, pushLog(err, helper.DBErr)
		}

		if data.T == 0 {
			return data, nil
		}
	}

	offset := (page - 1) * pageSize
	query, _, _ := dialect.From("f_vip").
		Select(colVip...).Where(ex).Offset(uint(offset)).Limit(uint(pageSize)).ToSQL()
	err := meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	// 拼装查询channel id的pids和查询cate_name的cids
	var (
		pids []string
		cids []string
	)
	for _, v := range data.D {
		pids = append(pids, v.PaymentID)
		cids = append(cids, v.CateID)
	}

	chs, _ := PaymentIDMapToChanID(pids)
	for k := range data.D {
		if v, ok := chs[data.D[k].PaymentID]; ok {
			data.D[k].ChannelID = v
		}
	}

	// 查询cate name
	cm, _ := cateByIDS(cids)
	for k := range data.D {
		if name, ok := cm[data.D[k].CateID]; ok {
			data.D[k].CateName = name
		}
	}

	data.S = pageSize
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
	err = meta.MerchantDB.Get(&id, query)
	if err != nil && err != sql.ErrNoRows {
		return pushLog(err, helper.DBErr)
	}

	if id != "" {
		return errors.New(helper.CateIDAndChannelIDErr)
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

func VipSet(id, state string, vip *Vip) error {

	// 上级通道关闭的时候不能开启
	if state == "1" {
		channel, err := ChanByID(vip.PaymentID)
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
		fmin, ok := validator.CheckFloatScope(vip.Fmin, channel.Fmin, channel.Fmax)
		if !ok {
			return errors.New(helper.AmountOutRange)
		}

		// check fmax scope
		fmax, ok := validator.CheckFloatScope(vip.Fmax, channel.Fmin, channel.Fmax)
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

	if vip.Flags == "2" {
		// 刷新代付(取款)缓存
		CreateAutomatic(vip.Vip)
	} else {
		// 刷新支付(存款)缓存
		Create(vip.Vip)
	}

	return nil
}

func VipByID(id string) (Vip, error) {

	vip := Vip{}

	query, _, _ := dialect.From("f_vip").Select(colVip...).Where(g.Ex{"id": id}).ToSQL()
	err := meta.MerchantDB.Get(&vip, query)
	if err != nil && err != sql.ErrNoRows {
		return vip, pushLog(err, helper.DBErr)
	}

	return vip, nil
}
