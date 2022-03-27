package model

import (
	"finance/contrib/helper"
	g "github.com/doug-martin/goqu/v9"
)

func PromoDetail(id string) (string, error) {

	var str string

	cond := g.Ex{
		"id":     id,
		"prefix": meta.Prefix,
	}
	query, _, _ := dialect.From("f_channel_type").Select("content").Where(cond).ToSQL()
	err := meta.MerchantDB.Get(&str, query)
	if err != nil {
		return "", pushLog(err, helper.DBErr)
	}

	return str, nil
}

func PromoUpdate(recs g.Record, id string) error {

	ex := g.Ex{
		"id":     id,
		"prefix": meta.Prefix,
	}
	query, _, _ := dialect.Update("f_channel_type").Set(recs).Where(ex).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return nil
}
