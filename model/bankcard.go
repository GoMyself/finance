package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"finance/contrib/validator"

	g "github.com/doug-martin/goqu/v9"
)

type BankCard struct {
	ID            string  `db:"id"              json:"id"             ` //
	ChannelBankID string  `db:"channel_bank_id" json:"channel_bank_id"` //
	Name          string  `db:"name"            json:"name"           ` //
	CardNo        string  `db:"card_no"         json:"card_no"        ` //
	RealName      string  `db:"real_name"       json:"real_name"      ` //
	BankAddr      string  `db:"bank_addr"       json:"bank_addr"      ` //
	State         int     `db:"state"           json:"state"          ` //
	MaxAmount     float64 `db:"max_amount"      json:"max_amount"     ` //
	Remark        string  `db:"remark"          json:"remark"         ` //
	Money         float64 `db:"-"               json:"money"          ` // 今日存款金额
	Prefix        string  `db:"prefix" json:"prefix"`
}

// BankCardListForDeposit 银行卡信息 线下转卡 订单列表
type BankCardListForDeposit struct {
	ID       string `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	CardNo   string `db:"card_no" json:"card_no"`
	RealName string `db:"real_name" json:"real_name"`
	BankAddr string `db:"bank_addr" json:"bank_addr"`
}

func BankCardInsert(recs BankCard) error {

	if meta.Lang == "vn" {
		if !validator.CheckStringVName(recs.RealName) {
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
func BankCardList(ex g.Ex) ([]BankCard, error) {

	var data []BankCard

	ex["prefix"] = meta.Prefix

	query, _, _ := dialect.From("f_bankcards").Select(colBankCard...).Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&data, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}

func BankCardByID(id string) (BankCard, error) {

	var bc BankCard
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

func BankCardByCol(col, val string) (BankCard, error) {

	var bc BankCard
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

/*
func BankCardOpenCondition(id, channelBankID string, max float64) error {

	key := BankCardTotalKey(id)
	rs, err := meta.MerchantRedis.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return pushLog(err, helper.RedisErr)
	}

	money := decimal.Zero
	if err != redis.Nil {
		money, err = decimal.NewFromString(rs)
		if err != nil {
			return errors.New(helper.FormatErr)
		}
	}

	if money.GreaterThanOrEqual(decimal.NewFromFloat(max)) {
		return errors.New(helper.ChangeDepositLimitBeforeActive)
	}

	ex := g.Ex{
		"channel_bank_id": channelBankID,
		"state":           1,
		"id":              g.Op{"neq": id},
		"prefix":          meta.Prefix,
	}

	b, err := BankCardExistByEx(ex)
	if err != nil {
		return err
	}

	if b {
		return errors.New(helper.OnlyOneBankcardActivePerBank)
	}

	return nil
}
*/
/*
func BankCardTotalKey(bankID string) string {
	timeStr := time.Now().In(loc).Format("2006-01-02")
	return fmt.Sprintf("BT:%s:%s", bankID, timeStr)
}
*/
