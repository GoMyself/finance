package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"fmt"
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
	Prefix       string `db:"prefix"`        //站点前缀
	OperationNo  string `db:"operation_no"`  //三方单号
}

// MemberCache 通过用户名获取用户在redis中的数据
func MemberCache(fctx *fasthttp.RequestCtx) (Member, error) {

	m := Member{}
	name := string(fctx.UserValue("token").([]byte))
	if name == "" {
		return m, errors.New(helper.UsernameErr)
	}

	//pipe := meta.MerchantRedis.TxPipeline()
	//defer pipe.Close()
	//
	//exist := pipe.Exists(ctx, name)
	//rs := pipe.HMGet(ctx, name, "uid", "username", "realname_hash", "state", "top_uid", "top_name", "parent_uid", "parent_name", "level")
	//
	//_, err := pipe.Exec(ctx)
	//if err != nil {
	//	return m, pushLog(err, helper.RedisErr)
	//}
	//
	//if exist.Val() == 0 {
	//	return m, errors.New(helper.UsernameErr)
	//}
	//
	//if err = rs.Scan(&m); err != nil {
	//	return m, pushLog(rs.Err(), helper.RedisErr)
	//}
	t := dialect.From("tbl_members")
	query, _, _ := t.Select(colsMember...).Where(g.Ex{"username": name, "prefix": meta.Prefix}).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&m, query)
	if err != nil && err != sql.ErrNoRows {
		return m, pushLog(err, helper.DBErr)
	}

	if err == sql.ErrNoRows {
		return m, errors.New(helper.UsernameErr)
	}

	return m, nil
}

type MemberInfo struct {
	UID                 string `db:"uid" json:"uid"`
	Username            string `db:"username" json:"username"`                           //会员名
	Password            string `db:"password" json:"password"`                           //密码
	RealnameHash        uint64 `db:"realname_hash" json:"realname_hash"`                 //真实姓名哈希
	EmailHash           uint64 `db:"email_hash" json:"email_hash"`                       //邮件地址哈希
	PhoneHash           uint64 `db:"phone_hash" json:"phone_hash"`                       //电话号码哈希
	Prefix              string `db:"prefix" json:"prefix"`                               //站点前缀
	WithdrawPwd         uint64 `db:"withdraw_pwd" json:"withdraw_pwd"`                   //取款密码哈希
	Regip               string `db:"regip" json:"regip"`                                 //注册IP
	RegDevice           string `db:"reg_device" json:"reg_device"`                       //注册设备号
	RegUrl              string `db:"reg_url" json:"reg_url"`                             //注册链接
	CreatedAt           uint32 `db:"created_at" json:"created_at"`                       //注册时间
	LastLoginIp         string `db:"last_login_ip" json:"last_login_ip"`                 //最后登陆ip
	LastLoginAt         uint32 `db:"last_login_at" json:"last_login_at"`                 //最后登陆时间
	SourceId            uint8  `db:"source_id" json:"source_id"`                         //注册来源 1 pc 2h5 3 app
	FirstDepositAt      uint32 `db:"first_deposit_at" json:"first_deposit_at"`           //首充时间
	FirstDepositAmount  string `db:"first_deposit_amount" json:"first_deposit_amount"`   //首充金额
	SecondDepositAt     uint32 `db:"second_deposit_at" json:"second_deposit_at"`         //二存时间
	SecondDepositAmount string `db:"second_deposit_amount" json:"second_deposit_amount"` //二充金额
	FirstBetAt          uint32 `db:"first_bet_at" json:"first_bet_at"`                   //首投时间
	FirstBetAmount      string `db:"first_bet_amount" json:"first_bet_amount"`           //首投金额
	TopUid              string `db:"top_uid" json:"top_uid"`                             //总代uid
	TopName             string `db:"top_name" json:"top_name"`                           //总代代理
	ParentUid           string `db:"parent_uid" json:"parent_uid"`                       //上级uid
	ParentName          string `db:"parent_name" json:"parent_name"`                     //上级代理
	BankcardTotal       uint8  `db:"bankcard_total" json:"bankcard_total"`               //用户绑定银行卡的数量
	LastLoginDevice     string `db:"last_login_device" json:"last_login_device"`         //最后登陆设备
	LastLoginSource     int    `db:"last_login_source" json:"last_login_source"`         //上次登录设备来源:1=pc,2=h5,3=ios,4=andriod
	Remarks             string `db:"remarks" json:"remarks"`                             //备注
	State               uint8  `db:"state" json:"state"`                                 //状态 1正常 2禁用
	Balance             string `db:"balance" json:"balance"`                             //余额
	LockAmount          string `db:"lock_amount" json:"lock_amount"`                     //锁定金额
	Commission          string `db:"commission" json:"commission"`                       //佣金
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

//获取用户 首存 和 二存
func GetUserDeposit(uid string) (MBBalance, error) {

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

func MembersByUsernames(username []string) ([]Member, error) {

	var members []Member
	ex := g.Ex{"username": username, "prefix": meta.Prefix}
	query, _, err := dialect.From("tbl_members").Select(colsMember...).Where(ex).ToSQL()
	err = meta.MerchantDB.Select(&members, query)
	if err != nil {
		return members, pushLog(err, helper.DBErr)
	}

	return members, err
}

func getBalanceByUids(uids []string) ([]MBBalance, error) {
	var (
		balances []MBBalance
		query    string
	)

	ex := g.Ex{
		"uid":    uids,
		"prefix": meta.Prefix,
	}
	query, _, _ = dialect.From("tbl_members").Select("balance", "uid", "lock_amount").Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&balances, query)
	if err != nil {
		return balances, pushLog(err, helper.DBErr)
	}

	return balances, nil
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

// 查询用户单条数据
func MemberFindOne(name string) (MemberInfo, error) {

	m := MemberInfo{}

	t := dialect.From("tbl_members")
	query, _, _ := t.Select(colsMemberInfo...).Where(g.Ex{"username": name}).Limit(1).ToSQL()
	fmt.Printf("MemberFindOne : %v\n", query)
	err := meta.MerchantDB.Get(&m, query)
	if err != nil {
		return m, err
	}

	return m, nil
}
