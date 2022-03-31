package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"fmt"

	g "github.com/doug-martin/goqu/v9"
	"github.com/go-redis/redis/v8"
)

type Category struct {
	ID         string `db:"id" json:"id"`
	Name       string `db:"name" json:"name"`
	MerchantId string `db:"merchant_id" json:"merchant_id"`
	State      string `db:"state" json:"state"`
	Comment    string `db:"comment" json:"comment"`
	CreatedAt  int64  `db:"created_at" json:"created_at"`
	Prefix     string `db:"prefix" json:"prefix"`
}

// CategoryData 财务管理-渠道管理-列表 response structure
type CategoryData struct {
	D []Category `json:"d"`
	T int64      `json:"t"`
	S uint16     `json:"s"`
}

// CateIDAndName 渠道id和name
type CateIDAndName struct {
	ID   string `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}

func CateList(name, all string, page, pageSize uint16) (CategoryData, error) {

	var data CategoryData

	// 新增渠道时用
	if all == "1" {
		cond := g.Ex{
			"state":  "1",
			"prefix": meta.Prefix,
		}
		query, _, _ := dialect.From("f_category").Select(colCate...).Where(cond).Order(g.C("created_at").Asc()).ToSQL()
		err := meta.MerchantDB.Select(&data.D, query)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}

		return data, nil
	}

	ex := g.Ex{
		"prefix": meta.Prefix,
	}

	if name != "" {
		ex["name"] = name
	}

	if page == 1 {
		query, _, _ := dialect.From("f_category").Select(g.COUNT(1)).Where(ex).ToSQL()
		err := meta.MerchantDB.Get(&data.T, query)
		if err != nil && err != sql.ErrNoRows {
			return data, pushLog(err, helper.DBErr)
		}

		if data.T == 0 {
			return data, nil
		}
	}

	offset := (page - 1) * pageSize
	query, _, _ := dialect.From("f_category").
		Select(colCate...).Where(ex).Order(g.C("created_at").Desc()).Offset(uint(offset)).Limit(uint(pageSize)).ToSQL()
	err := meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	data.S = pageSize
	return data, nil
}

func CateWithdrawList(amount float64) (CategoryData, error) {

	var data CategoryData

	ex := g.Ex{
		"channel_id": "7",
		"state":      "1",
		"prefix":     meta.Prefix,
	}
	if amount != 0 {
		ex["fmin"] = g.Op{"lte": amount}
		ex["fmax"] = g.Op{"gte": amount}
	}

	var pids []string
	query, _, _ := dialect.From("f_payment").Select("cate_id").Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&pids, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	if len(pids) == 0 {
		return data, nil
	}

	ex = g.Ex{
		"id":     pids,
		"state":  "1",
		"prefix": meta.Prefix,
	}
	query, _, _ = dialect.From("f_category").Select(colCate...).Where(ex).Order(g.C("created_at").Desc()).ToSQL()
	err = meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}

func CateInsert(param map[string]string) error {

	// 商户id和渠道号唯一
	err := checkMidAndName(param["name"], "")
	if err != nil {
		return err
	}

	record := g.Record{
		"name":       param["name"],
		"comment":    param["comment"],
		"created_at": param["created_at"],
		"state":      "0", // 状态默认是0:关闭
		"prefix":     meta.Prefix,
	}
	query, _, _ := dialect.Insert("f_category").Rows(record).ToSQL()
	_, err = meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return nil
}

func CateUpdate(param map[string]string) error {

	// 商户id和渠道号唯一
	err := checkMidAndName(param["name"], param["id"])
	if err != nil {
		return err
	}

	cate, err := CateByID(param["id"])
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	if len(cate.ID) == 0 {
		return errors.New(helper.RecordNotExistErr)
	}

	if cate.State == "1" {
		return errors.New(helper.UpdateMustCloseFirst)
	}

	record := g.Record{
		"name":    param["name"],
		"comment": param["comment"],
	}

	ex := g.Ex{
		"prefix": meta.Prefix,
		"id":     param["id"],
	}

	query, _, _ := dialect.Update("f_category").Set(record).Where(ex).ToSQL()
	_, err = meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return nil
}

func CateDelete(id string) error {

	cate, err := CateByID(id)
	if err != nil {
		return err
	}

	if len(cate.ID) == 0 {
		return errors.New(helper.RecordNotExistErr)
	}

	if cate.State == "1" {
		return errors.New(helper.DeleteMustCloseFirst)
	}

	// 渠道下级有通道就不能删除
	ch, _ := ChanByCateID(id)
	if ch.ID != "" {
		return errors.New(helper.CateHaveChannelDeleteErr)
	}

	ex := g.Ex{
		"prefix": meta.Prefix,
		"id":     id,
	}

	query, _, _ := dialect.Delete("f_category").Where(ex).ToSQL()
	_, err = meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return nil
}

// 设置渠道的状态 开启/关闭
func CateSet(id, state string) error {

	tx, err := meta.MerchantDB.Begin()
	if err != nil {
		return errors.New(helper.TransErr)
	}

	ex := g.Ex{
		"prefix": meta.Prefix,
		"id":     id,
	}

	query, _, _ := dialect.Update("f_category").Set(g.Record{"state": state}).Where(ex).ToSQL()
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return errors.New(helper.TransErr)
	}

	// 切换到关闭状态，旗下所有支付方式也将同时切换到关闭状态
	if state == "0" {
		ex = g.Ex{
			"prefix":  meta.Prefix,
			"cate_id": id,
		}
		query, _, _ = dialect.Update("f_payment").Set(g.Record{"state": state}).Where(ex).ToSQL()
		_, err = tx.Exec(query)
		if err != nil {
			_ = tx.Rollback()
			return errors.New(helper.TransErr)
		}

		query, _, _ = dialect.Update("f_vip").Set(g.Record{"state": state}).Where(ex).ToSQL()
		_, err = tx.Exec(query)
		if err != nil {
			_ = tx.Rollback()
			return errors.New(helper.TransErr)
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.New(helper.TransErr)
	}

	var levels []string
	ex = g.Ex{
		"prefix":  meta.Prefix,
		"cate_id": id,
	}
	query, _, _ = dialect.From("f_vip").Select("vip").Where(ex).GroupBy("vip").ToSQL()
	err = meta.MerchantDB.Select(&levels, query)
	if err == nil {
		for _, level := range levels {
			Create(level)
		}
	}

	cateToRedis()
	return nil
}

// 三方渠道
func CateByID(id string) (Category, error) {

	var cate Category
	ex := g.Ex{
		"prefix": meta.Prefix,
		"id":     id,
	}
	query, _, _ := dialect.From("f_category").Select(colCate...).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&cate, query)
	if err != nil && err != sql.ErrNoRows {
		return cate, pushLog(err, helper.DBErr)
	}

	return cate, nil
}

// 商户号和渠道名称唯一
func checkMidAndName(name, id string) error {

	var cate Category

	// 新增
	if id == "" {
		ex := g.Ex{
			"prefix": meta.Prefix,
			"name":   name,
		}
		query, _, _ := dialect.From("f_category").Select(colCate...).Where(ex).ToSQL()
		err := meta.MerchantDB.Get(&cate, query)

		switch err {
		case sql.ErrNoRows:
			return nil
		case nil:
			return errors.New(helper.MerchantIDOrCateNameExist)
		}

		return pushLog(err, helper.DBErr)
	}

	// 编辑
	ex := g.Ex{
		"name":   name,
		"id":     g.Op{"neq": id},
		"prefix": meta.Prefix,
	}
	query, _, _ := dialect.From("f_category").Select(colCate...).
		Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&cate, query)
	if err != nil && err != sql.ErrNoRows {
		return errors.New(helper.DBErr)
	}

	if err != sql.ErrNoRows {
		if cate.ID != id {
			return errors.New(helper.MerchantIDOrCateNameExist)
		}
	}

	return nil
}

func cateByIDS(ids []string) (map[string]string, error) {

	var (
		data []Category
		res  = map[string]string{}
	)

	if len(ids) == 0 {
		return res, nil
	}

	ex := g.Ex{
		"id":     g.Op{"in": ids},
		"prefix": meta.Prefix,
	}
	query, _, _ := dialect.From("f_category").Select(colCate...).Where(ex).GroupBy("id").ToSQL()
	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return res, pushLog(err, helper.DBErr)
	}

	for _, v := range data {
		if _, ok := res[v.ID]; !ok {
			res[v.ID] = v.Name
		}
	}

	return res, nil
}

func cateToRedis() {
	var cate []Category
	ex := g.Ex{
		"prefix": meta.Prefix,
	}
	query, _, _ := dialect.From("f_category").Select(colCate...).Where(ex).Order(g.C("id").Asc()).ToSQL()
	fmt.Println(query)
	err := meta.MerchantDB.Select(&cate, query)
	if err != nil || len(cate) < 1 {
		return
	}

	res := map[string]string{}
	for _, v := range cate {
		res[v.ID] = v.Name
		fmt.Printf(v.ID + ":" + v.Name)
	}
	b, err := helper.JsonMarshal(res)
	if err != nil {
		return
	}

	key := "f:category"
	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	pipe.Unlink(ctx, key)
	pipe.Set(ctx, key, string(b), 0)
	_, _ = pipe.Exec(ctx)
}

func CateListRedis() string {

	res, err := meta.MerchantRedis.Get(ctx, "f:category").Result()
	if err == redis.Nil || err != nil {
		return "{}"
	}

	return res
}

// CateIDAndNameByCIDS 通过cid查询渠道id和渠道名
func CateIDAndNameByCIDS(cids []string) (map[string]CateIDAndName, error) {

	var (
		data []CateIDAndName
		res  = make(map[string]CateIDAndName)
	)

	if len(cids) == 0 {
		return res, nil
	}

	ex := g.Ex{
		"id":     g.Op{"in": cids},
		"prefix": meta.Prefix,
	}
	query, _, _ := dialect.From("f_category").Select("id", "name").Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return res, pushLog(err, helper.DBErr)
	}

	for _, v := range data {
		if _, ok := res[v.ID]; !ok {
			res[v.ID] = v
		}
	}

	return res, nil
}
