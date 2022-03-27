package model

import (
	"errors"
	"finance/contrib/helper"
	g "github.com/doug-martin/goqu/v9"
	"github.com/shopspring/decimal"
	"github.com/valyala/fasthttp"
)

type Member struct {
	UID          string `db:"uid"                  json:"uid"                  redis:"uid"                 ` //
	Username     string `db:"username"             json:"username"             redis:"username"            ` //
	Prefix       string `db:"prefix"               json:"prefix"               redis:"prefix"              ` //
	RealnameHash uint64 `db:"realname_hash"        json:"realname_hash"        redis:"realname_hash"       ` //
	State        int    `db:"state"                json:"state"                redis:"state"               ` //状态 1正常 2禁用
	TopUID       string `db:"top_uid"              json:"top_uid"              redis:"top_uid"             ` // 总代uid
	TopName      string `db:"top_name"             json:"top_name"             redis:"top_name"            ` // 总代代理
	ParentUID    string `db:"parent_uid"           json:"parent_uid"           redis:"parent_uid"          ` //  上级uid
	ParentName   string `db:"parent_name"          json:"parent_name"          redis:"parent_name"         ` // 上级代理
	Level        int    `db:"level"                json:"level"                redis:"level"               ` // 上级代理
}

type MBBalance struct {
	UID        string  `db:"uid"         json:"uid"         redis:"uid"        ` //主键ID
	Balance    float64 `db:"balance"     json:"balance"     redis:"balance"    ` //余额
	LockAmount float64 `db:"lock_amount" json:"lock_amount" redis:"lock_amount"` //锁定额度
}

//账变表
type memberTransaction struct {
	AfterAmount  string `db:"after_amount"`  //账变后的金额
	Amount       string `db:"amount"`        //用户填写的转换金额
	BeforeAmount string `db:"before_amount"` //账变前的金额
	BillNo       string `db:"bill_no"`       //转账|充值|提现ID
	CashType     int    `db:"cash_type"`     //0:转入1:转出2:转入失败补回3:转出失败扣除4:存款5:提现
	CreatedAt    int64  `db:"created_at"`    //
	ID           string `db:"id"`            //
	UID          string `db:"uid"`           //用户ID
	Username     string `db:"username"`      //用户名
	Prefix       string `db:"prefix"`
}

// MemberCache 通过用户名获取用户在redis中的数据
func MemberCache(fCtx *fasthttp.RequestCtx) (Member, error) {

	m := Member{}
	name := string(fCtx.UserValue("token").([]byte))
	if name == "" {
		return m, errors.New(helper.UsernameErr)
	}

	ex := g.Ex{
		"username": name,
		"prefix":   meta.Prefix,
	}
	query, _, _ := dialect.From("tbl_members").Select(colsMember...).Where(ex).ToSQL()
	err := meta.MerchantDB.Get(&m, query)
	if err != nil {
		return m, pushLog(err, helper.DBErr)
	}

	return m, nil
}

func MemberGetByName(username string) (Member, error) {

	m := Member{}

	t := dialect.From("tbl_members")
	query, _, _ := t.Select(colsMember...).Where(g.Ex{"username": username, "prefix": meta.Prefix}).ToSQL()
	err := meta.MerchantDB.Get(&m, query)
	if err != nil {
		return m, pushLog(err, helper.DBErr)
	}

	return m, nil
}

//读取余额
func GetBalanceDB(uid string) (MBBalance, error) {

	var (
		balance MBBalance
		query   string
	)

	ex := g.Ex{
		"uid":    uid,
		"prefix": meta.Prefix,
	}
	query, _, _ = dialect.From("tbl_members").Select("balance", "uid", "lock_amount").Where(ex).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&balance, query)
	if err != nil {
		return balance, pushLog(err, helper.DBErr)
	}

	return balance, nil
}

func membersUidByUsername(username []string) (map[string]string, error) {

	var users []Member
	ex := g.Ex{"username": username, "prefix": meta.Prefix}
	query, _, _ := dialect.From("tbl_members").Select("uid", "username").Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&users, query)
	if err != nil {
		return nil, pushLog(err, helper.DBErr)
	}

	m := map[string]string{}
	for _, v := range users {
		m[v.Username] = v.UID
	}

	return m, nil
}

func MemberByUsername(username string) (Member, error) {

	var member Member
	ex := g.Ex{"username": username, "prefix": meta.Prefix}
	query, _, err := dialect.From("tbl_members").Select(colsMember...).Where(ex).ToSQL()
	err = meta.MerchantDB.Get(&member, query)
	if err != nil {
		return member, pushLog(err, helper.DBErr)
	}

	return member, err
}

// BalanceIsEnough 检查中心钱包余额是否充足
func BalanceIsEnough(uid string, amount decimal.Decimal) (decimal.Decimal, error) {

	balance, err := GetBalanceDB(uid)
	if err != nil {
		return decimal.NewFromFloat(balance.Balance), err
	}

	if decimal.NewFromFloat(balance.Balance).Sub(amount).IsNegative() {
		return decimal.NewFromFloat(balance.Balance), errors.New(helper.LackOfBalance)
	}

	return decimal.NewFromFloat(balance.Balance), nil
}

func MemberLevelByUID(uids []string) (map[string]int, error) {

	var levels []Member
	ex := g.Ex{"uid": uids}
	query, _, _ := dialect.From("tbl_members").Select("uid", "level").Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&levels, query)
	if err != nil {
		return nil, pushLog(err, helper.DBErr)
	}

	res := make(map[string]int)
	for _, v := range levels {
		res[v.UID] = v.Level
	}

	return res, nil
}
