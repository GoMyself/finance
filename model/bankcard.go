package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"finance/contrib/validator"

	g "github.com/doug-martin/goqu/v9"
)

type Bankcard_t struct {
	Id                    string `db:"id" json:"id"`
	ChannelBankId         string `db:"channel_bank_id" json:"channel_bank_id"`                 // t_channel_bank的id
	BanklcardName         string `db:"banklcard_name" json:"banklcard_name"`                   // 银行名称
	BanklcardNo           string `db:"banklcard_no" json:"banklcard_no"`                       // 银行卡号
	AccountName           string `db:"account_name" json:"account_name"`                       // 持卡人姓名
	BankcardAddr          string `db:"bankcard_addr" json:"bankcard_addr"`                     // 开户行地址
	State                 string `db:"state" json:"state"`                                     // 状态：0 关闭  1 开启
	Remark                string `db:"remark" json:"remark"`                                   // 备注
	Prefix                string `db:"prefix" json:"prefix"`                                   // 商户前缀
	DailyMaxAmount        string `db:"daily_max_amount" json:"daily_max_amount"`               // 当天最大收款限额
	DailyFinishAmount     string `db:"daily_finish_amount" json:"daily_finish_amount"`         // 当天已收款总额
	TotalMaxAmount        string `db:"total_max_amount" json:"total_max_amount"`               // 累计最大收款限额
	TotalFinishAmountCopy string `db:"total_finish_amountCopy" json:"total_finish_amountCopy"` // 累计已收款总额
}

// BankCardListForDeposit 银行卡信息 线下转卡 订单列表
type BankCardListForDeposit struct {
	ID       string `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	CardNo   string `db:"card_no" json:"card_no"`
	RealName string `db:"real_name" json:"real_name"`
	BankAddr string `db:"bank_addr" json:"bank_addr"`
}

func BankCardInsert(recs Bankcard_t) error {

	if meta.Lang == "vn" {
		if !validator.CheckStringVName(recs.AccountName) {
			return errors.New(helper.ParamErr)
		}
	}

	recs.Prefix = meta.Prefix

	query, _, _ := dialect.Insert("f_bankcards").Rows(recs).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return nil
}

//BankCardDelete 删除银行卡
func BankCardDelete(id string) error {

	ex := g.Ex{
		"id": id,
	}
	query, _, _ := dialect.Delete("f_bankcards").Where(ex).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return nil
}

//BankCardList 银行卡列表
func BankCardList(ex g.Ex) ([]Bankcard_t, error) {

	var data []Bankcard_t

	ex["prefix"] = meta.Prefix

	query, _, _ := dialect.From("f_bankcards").Select(colBankCard...).Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}

func BankCardByID(id string) (Bankcard_t, error) {

	var bc Bankcard_t
	ex := g.Ex{
		"id": id,
	}
	query, _, _ := dialect.From("f_bankcards").Select(colBankCard...).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&bc, query)
	if err != nil && err != sql.ErrNoRows {
		return bc, pushLog(err, helper.DBErr)
	}

	if err == sql.ErrNoRows {
		return bc, errors.New(helper.BankCardNotExist)
	}

	return bc, nil
}

func BankCardUpdate(id string, record g.Record) error {

	ex := g.Ex{
		"id": id,
	}
	query, _, _ := dialect.Update("f_bankcards").Set(record).Where(ex).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return nil
}

func BankCardByCol(col, val string) (Bankcard_t, error) {

	var bc Bankcard_t
	ex := g.Ex{
		col:      val,
		"prefix": meta.Prefix,
	}
	query, _, _ := dialect.From("f_bankcards").Select(colBankCard...).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&bc, query)
	if err != nil && err != sql.ErrNoRows {
		return bc, pushLog(err, helper.DBErr)
	}

	if err == sql.ErrNoRows {
		return bc, errors.New(helper.RecordNotExistErr)
	}

	return bc, nil
}

func BankCardExistByEx(ex g.Ex) (bool, error) {

	var bc int
	query, _, _ := dialect.From("f_bankcards").Where(ex).Select(g.COUNT("id")).ToSQL()
	err := meta.MerchantDB.Get(&bc, query)
	if err != nil {
		return false, pushLog(err, helper.DBErr)
	}

	return bc > 0, nil
}
