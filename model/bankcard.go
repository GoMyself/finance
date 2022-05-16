package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"finance/contrib/validator"
	"fmt"

	g "github.com/doug-martin/goqu/v9"
)

type Bankcard_t struct {
	Id                string `db:"id" json:"id" json:"id"`
	ChannelBankId     string `db:"channel_bank_id" json:"bank_id"`                 // t_channel_bank的id
	BanklcardName     string `db:"banklcard_name" json:"banklcard_name"`           // 银行名称
	BanklcardNo       string `db:"banklcard_no" json:"banklcard_no"`               // 银行卡号
	AccountName       string `db:"account_name" json:"account_name"`               // 持卡人姓名
	BankcardAddr      string `db:"bankcard_addr" json:"bankcard_addr"`             // 开户行地址
	State             string `db:"state" json:"state"`                             // 状态：0 关闭  1 开启
	Remark            string `db:"remark" json:"remark"`                           // 备注
	Prefix            string `db:"prefix" json:"prefix"`                           // 商户前缀
	DailyMaxAmount    string `db:"daily_max_amount" json:"daily_max_amount"`       // 当天最大收款限额
	DailyFinishAmount string `db:"daily_finish_amount" json:"daily_finish_amount"` // 当天已收款总额
	TotalMaxAmount    string `db:"total_max_amount" json:"total_max_amount"`       // 累计最大收款限额
	TotalFinishAmount string `db:"total_finish_amount" json:"total_finish_amount"` // 累计已收款总额
	Flags             string `db:"flags" json:"flags"`                             // 累计已收款总额
}

// BankCardListForDeposit 银行卡信息 线下转卡 订单列表
type BankCardListForDeposit struct {
	ID       string `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	CardNo   string `db:"card_no" json:"card_no"`
	RealName string `db:"real_name" json:"real_name"`
	BankAddr string `db:"bank_addr" json:"bank_addr"`
}

func BankCardBackend() (Bankcard_t, error) {

	bc := Bankcard_t{}
	key := "offlineBankcard"
	res, err := meta.MerchantRedis.RPopLPush(ctx, key, key).Result()
	if err != nil {
		return bc, errors.New(helper.RecordNotExistErr)
	}

	helper.JsonUnmarshal([]byte(res), &bc)
	return bc, nil
}

func BankCardUpdateCache() error {

	key := "offlineBankcard"
	ex := g.Ex{
		"state": "1",
		"flags": "1",
	}
	res, err := BankCardList(ex)
	if err != nil {
		fmt.Println("BankCardUpdateCache err = ", err)
		return err
	}

	if len(res) == 0 {
		fmt.Println("BankCardUpdateCache len(res) = 0")
		meta.MerchantRedis.Unlink(ctx, key).Err()
		return nil
	}

	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	pipe.Unlink(ctx, key)
	for _, v := range res {
		val, err := helper.JsonMarshal(v)
		if err != nil {
			continue
		}
		pipe.LPush(ctx, key, string(val))
	}
	pipe.Persist(ctx, key)

	_, err = pipe.Exec(ctx)
	if err != nil {
		fmt.Println("BankCardUpdateCache pipe.Exec = ", err)
		return errors.New(helper.RedisErr)
	}

	return nil
}

func BankCardInsert(recs Bankcard_t, code string) error {

	if meta.Lang == "vn" {
		if !validator.CheckStringVName(recs.AccountName) {
			return errors.New(helper.ParamErr)
		}
	}

	recs.Prefix = meta.Prefix

	query, _, _ := dialect.Insert("f_bankcards").Rows(recs).ToSQL()
	fmt.Println("BankCardInsert query = ", query)
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
	BankCardUpdateCache()
	return nil
}

func BankCardByCol(val string) (Bankcard_t, error) {

	var bc Bankcard_t
	ex := g.Ex{
		"banklcard_no": val,
		"prefix":       meta.Prefix,
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
