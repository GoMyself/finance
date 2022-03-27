package model

import (
	"database/sql"
	"finance/contrib/helper"
	g "github.com/doug-martin/goqu/v9"
)

type MemberBankCard struct {
	ID           string `db:"id" json:"id"`
	UID          string `db:"uid" json:"uid"`
	Username     string `db:"username" json:"username"`
	BankAddress  string `db:"bank_address" json:"bank_address"`
	BankID       string `db:"bank_id" json:"bank_id"`
	BankBranch   string `db:"bank_branch_name" json:"bank_branch_name"`
	State        int    `db:"state" json:"state"`
	BankcardHash string `db:"bank_card_hash" json:"bank_card_hash"`
	CreatedAt    uint64 `db:"created_at" json:"created_at"`
}

func MemberBankcardList(ex g.Ex) ([]MemberBankCard, error) {

	var data []MemberBankCard
	ex["prefix"] = meta.Prefix
	t := dialect.From("tbl_member_bankcard")
	query, _, _ := t.Select(colsMemberBankcard...).Where(ex).Order(g.C("created_at").Desc()).ToSQL()
	err := meta.MerchantDB.Select(&data, query)
	if err != nil && err != sql.ErrNoRows {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}
