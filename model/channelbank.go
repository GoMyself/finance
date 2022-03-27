package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	g "github.com/doug-martin/goqu/v9"
)

type ChannelBanks struct {
	ID        string `db:"id" json:"id"`
	BankID    int64  `db:"bank_id" json:"bank_id"`
	CateID    string `db:"cate_id" json:"cate_id"`
	CateName  string `db:"-" json:"cate_name"`
	ChannelID string `db:"-" json:"channel_id"`
	PaymentID string `db:"payment_id" json:"payment_id"`
	Name      string `db:"name" json:"name"`
	Code      string `db:"code" json:"code"`
	Sort      int64  `db:"sort" json:"sort"`
	State     string `db:"state" json:"state"`
}

// ChannelBankData 财务管理-通道银行管理-列表 response structure
type ChannelBankData struct {
	D []ChannelBanks `json:"d"`
	T int64          `json:"t"`
	S uint16         `json:"s"`
}

func ChannelBankList(cateID, chanID string, page, pageSize uint16) (ChannelBankData, error) {

	var data ChannelBankData

	ex := g.Ex{
		"prefix": meta.Prefix,
	}
	if cateID != "0" {
		ex["cate_id"] = cateID
	}

	if chanID != "0" {
		var ids []string
		query, _, _ := dialect.From("f_payment").Select("id").GroupBy("id").
			Where(g.Ex{"channel_id": chanID, "prefix": meta.Prefix}).ToSQL()
		err := meta.MerchantDB.Select(&ids, query)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}

		if len(ids) == 0 {
			return data, nil
		}

		ex["payment_id"] = ids
	}

	if page == 1 {
		query, _, _ := dialect.From("f_channel_banks").Select(g.COUNT(1)).Where(ex).ToSQL()
		err := meta.MerchantDB.Get(&data.T, query)
		if err != nil && err != sql.ErrNoRows {
			return data, pushLog(err, helper.DBErr)
		}

		if data.T == 0 {
			return data, nil
		}
	}

	offset := (page - 1) * pageSize
	query, _, _ := dialect.From("f_channel_banks").Select(colChannelBank...).
		Where(ex).Order(g.C("sort").Asc()).Offset(uint(offset)).Limit(uint(pageSize)).ToSQL()
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

func ChannelBankInsert(param map[string]string) error {

	// check cate id and channel id
	channel, err := ChanByCateAndChan(param["cate_id"], param["channel_id"])
	if err != nil {
		return errors.New(helper.CateIDAndChannelIDErr)
	}

	if len(channel.ID) == 0 {
		return errors.New(helper.ChannelNotExist)
	}

	// 同一个payment id的code和name分别唯一
	err = checkBankCodeNameUnique(channel.ID, param["code"], param["name"], "")
	if err != nil {
		return err
	}

	record := g.Record{
		"id":         param["id"],
		"bank_id":    param["bank_id"],
		"cate_id":    param["cate_id"],
		"payment_id": channel.ID,
		"name":       param["name"],
		"state":      "0",
		"code":       param["code"],
		"sort":       param["sort"],
		"prefix":     meta.Prefix,
	}
	query, _, _ := dialect.Insert("f_channel_banks").Rows(record).ToSQL()
	_, err = meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	// _ = cacheRefreshPaymentBanks(channel.ID)

	return nil
}

func ChannelBankUpdate(param map[string]string) error {

	bank, err := ChannelBankByID(param["id"])
	if err != nil {
		return err
	}

	if len(bank.ID) == 0 {
		return errors.New(helper.RecordNotExistErr)
	}

	if bank.State == "1" {
		return errors.New(helper.UpdateMustCloseFirst)
	}

	// 同一个payment id的code和name分别唯一
	err = checkBankCodeNameUnique(bank.PaymentID, param["code"], param["name"], bank.ID)
	if err != nil {
		return err
	}

	record := g.Record{
		"bank_id": param["bank_id"],
		"name":    param["name"],
		"code":    param["code"],
		"sort":    param["sort"],
	}

	query, _, _ := dialect.Update("f_channel_banks").Set(record).Where(g.Ex{"id": param["id"]}).ToSQL()
	_, err = meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	// _ = cacheRefreshPaymentBanks(param["payment_id"])

	return nil
}

func ChannelBankDelete(id string) error {

	bank, err := ChannelBankByID(id)
	if err != nil {
		return err
	}

	if len(bank.ID) == 0 {
		return errors.New(helper.RecordNotExistErr)
	}

	if bank.State == "1" {
		return errors.New(helper.DeleteMustCloseFirst)
	}

	query, _, _ := dialect.Delete("f_channel_banks").Where(g.Ex{"id": id}).ToSQL()
	_, err = meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	// _ = cacheRefreshPaymentBanks(bank.PaymentID)

	return nil
}

func ChannelBankSet(id, state, paymentID string) error {

	query, _, _ := dialect.Update("f_channel_banks").Set(g.Record{"state": state}).Where(g.Ex{"id": id}).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	_ = CacheRefreshPaymentBanks(paymentID)

	return nil
}

// 渠道所支持的银行
func ChannelBankByID(id string) (ChannelBanks, error) {

	chb := ChannelBanks{}
	query, _, _ := dialect.From("f_channel_banks").Select(colChannelBank...).Where(g.Ex{"id": id}).ToSQL()
	err := meta.MerchantDB.Get(&chb, query)
	if err != nil && err != sql.ErrNoRows {
		return chb, pushLog(err, helper.DBErr)
	}

	return chb, nil
}

// 同一个payment id的code和name分别唯一
func checkBankCodeNameUnique(pid, code, name, id string) error {

	bank := ChannelBanks{}

	// 新增
	if id == "" {
		query, _, _ := dialect.From("f_channel_banks").Select(colChannelBank...).
			Where(g.ExOr{"code": code, "name": name}, g.Ex{"payment_id": pid, "prefix": meta.Prefix}).ToSQL()
		err := meta.MerchantDB.Get(&bank, query)

		switch err {
		case sql.ErrNoRows:
			return nil
		case nil:
			return errors.New(helper.BankNameOrCodeErr)
		}

		return pushLog(err, helper.DBErr)
	}

	// 编辑
	query, _, _ := dialect.From("f_channel_banks").Select(colChannelBank...).
		Where(g.ExOr{"code": code, "name": name}, g.Ex{"payment_id": pid, "id": g.Op{"neq": id}, "prefix": meta.Prefix}).ToSQL()
	err := meta.MerchantDB.Get(&bank, query)
	if err != nil && err != sql.ErrNoRows {
		return pushLog(err, helper.DBErr)
	}

	if err != sql.ErrNoRows {
		if bank.ID != id {
			return errors.New(helper.RecordExistErr)
		}
	}

	return nil
}
