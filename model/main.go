package model

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"fmt"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	g "github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/go-redis/redis/v8"
	"github.com/hprose/hprose-golang/v3/rpc/core"
	rpchttp "github.com/hprose/hprose-golang/v3/rpc/http"
	. "github.com/hprose/hprose-golang/v3/rpc/http/fasthttp"
	"github.com/jmoiron/sqlx"
	"github.com/olivere/elastic/v7"
	"github.com/shopspring/decimal"
	"github.com/spaolacci/murmur3"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"github.com/valyala/fastjson"
)

type MetaTable struct {
	MerchantDB    *sqlx.DB
	MerchantTD    *sqlx.DB
	MerchantRedis *redis.Client
	ES            *elastic.Client
	Nats          *nats.Conn
	Program       string
	Prefix        string
	Lang          string
	Fcallback     string
	IsDev         bool
	EsPrefix      string
	MerchantInfo  map[string]string
	Finance       map[string]map[string]interface{}
}

var grpc_t struct {
	View       func(rctx context.Context, uid, field string) ([]string, error)
	Encrypt    func(rctx context.Context, uid string, data [][]string) error
	Decrypt    func(rctx context.Context, uid string, hide bool, field []string) (map[string]string, error)
	DecryptAll func(rctx context.Context, uids []string, hide bool, field []string) (map[string]map[string]string, error)

	CheckDepositFlow func(rctx context.Context, username string) bool
}

var (
	meta *MetaTable
	loc  *time.Location
	fc   *fasthttp.Client
	ctx  = context.Background()

	dialect              = g.Dialect("mysql")
	zero                 = decimal.NewFromInt(0)
	colTunnel            = helper.EnumFields(Tunnel_t{})
	colCate              = helper.EnumFields(Category{})
	colPayment           = helper.EnumFields(Payment_t{})
	colVip               = helper.EnumFields(Vip_t{})
	colWithdraw          = helper.EnumFields(Withdraw{})
	colChannelBank       = helper.EnumFields(ChannelBanks{})
	colsDeposit          = helper.EnumFields(Deposit{})
	colCreditLevel       = helper.EnumFields(CreditLevel{})
	colMemberCreditLevel = helper.EnumFields(MemberCreditLevel{})
	colMemberLock        = helper.EnumFields(MemberLock{})
	colBankCard          = helper.EnumFields(Bankcard_t{})
	colsWithdraw         = helper.EnumFields(Withdraw{})
	colsMember           = helper.EnumFields(Member{})
	colsMemberBankcard   = helper.EnumFields(MemberBankCard{})
	closChannelDevice    = helper.EnumFields(ChannelDevice{})
	colsMemberInfo       = helper.EnumFields(MemberInfo{})

	depositFields  = helper.EnumRedisFields(Deposit{})
	withdrawFields = helper.EnumRedisFields(Withdraw{})
)

var (
	paymentRoute  = map[string]Payment{}
	paymentLogTag = "payment_log"
	// 通过redis锁定提款订单的key
	depositOrderLockKey = "d:order:%s"
	// 通过redis锁定提款订单的key
	withdrawOrderLockKey = "w:order:%s"
	//usdt汇率 设置
	usdtKey = "usdt_rate"
)

//提现钱包
const (
	MemberWallet = 1 //用户中心钱包
	AgencyWallet = 2 //代理的佣金钱包
)

var defaultLevelWithdrawLimit = map[string]string{
	"count_remain":   "7",
	"max_remain":     "700000",
	"withdraw_count": "7",
	"withdraw_max":   "700000",
}

func Constructor(mt *MetaTable, socks5, c string) {

	meta = mt

	if meta.Lang == "cn" {
		loc, _ = time.LoadLocation("Asia/Shanghai")
	} else if meta.Lang == "vn" || meta.Lang == "th" {
		loc, _ = time.LoadLocation("Asia/Bangkok")
	}

	meta.MerchantInfo = map[string]string{
		"6":  "W",
		"12": "Manual",
		"13": "USDT2",
		"17": "VTPAY",
		"18": "918PAY",
		"19": "P3PAY",
		"20": "DBPAY",
	}

	cateToRedis()

	rpchttp.RegisterHandler()
	RegisterTransport()

	client := core.NewClient(c)
	//client.Use(log.Plugin)

	client.UseService(&grpc_t)

	fc = &fasthttp.Client{
		MaxConnsPerHost: 60000,
		TLSConfig:       &tls.Config{InsecureSkipVerify: true},
		ReadTimeout:     time.Second * 10,
		WriteTimeout:    time.Second * 10,
	}

	if socks5 != "0.0.0.0" {
		fc.Dial = fasthttpproxy.FasthttpHTTPDialer(socks5)
	}

	NewPayment()
}

func pushLog(err error, code string) error {

	_, file, line, _ := runtime.Caller(1)
	paths := strings.Split(file, "/")
	l := len(paths)
	if l > 2 {
		file = paths[l-2] + "/" + paths[l-1]
	}
	path := fmt.Sprintf("%s:%d", file, line)

	ts := time.Now()
	id := helper.GenId()
	fields := g.Record{
		"id":       id,
		"content":  err.Error(),
		"project":  meta.Program,
		"flags":    code,
		"filename": path,
		"ts":       ts.In(loc).UnixMilli(),
	}

	query, _, _ := dialect.Insert("goerror").Rows(&fields).ToSQL()
	//fmt.Println(query)
	_, err1 := meta.MerchantTD.Exec(query)
	if err1 != nil {
		fmt.Println("insert goerror query = ", query)
		fmt.Println("insert goerror err = ", err1.Error())

	}

	note := fmt.Sprintf("Hệ thống lỗi %s", id)
	return errors.New(note)
}

func Close() {
	meta.MerchantTD.Close()
	_ = meta.MerchantDB.Close()
	_ = meta.MerchantRedis.Close()
	//meta.MQPool.Release()
}

func AdminToken(ctx *fasthttp.RequestCtx) (map[string]string, error) {

	b := ctx.UserValue("token").([]byte)

	var p fastjson.Parser

	data := map[string]string{}
	v, err := p.ParseBytes(b)
	if err != nil {
		return data, err
	}

	o, err := v.Object()
	if err != nil {
		return data, err
	}

	o.Visit(func(k []byte, v *fastjson.Value) {
		key := string(k)
		val, err := v.StringBytes()
		if err == nil {
			data[key] = string(val)
		}
	})

	return data, nil
}

func MurmurHash(str string, seed uint32) uint64 {

	h64 := murmur3.New64WithSeed(seed)
	h64.Write([]byte(str))
	v := h64.Sum64()
	h64.Reset()

	return v
}

// 获取admin的name
func AdminGetName(id string) (string, error) {

	var name string
	query, _, _ := dialect.From("tbl_admins").Select("name").Where(g.Ex{"id": id}).ToSQL()
	err := meta.MerchantDB.Get(&name, query)
	if err != nil && err != sql.ErrNoRows {
		return name, pushLog(err, helper.DBErr)
	}

	return name, nil
}

func PushMerchantNotify(format, applyName, username, amount string) error {

	msg := fmt.Sprintf(format, applyName, username, amount, applyName, username, amount, applyName, username, amount)
	msg = strings.TrimSpace(msg)
	err := meta.Nats.Publish(meta.Prefix+":merchant_notify", []byte(msg))
	fmt.Printf("Nats send a message: %s\n", msg)
	if err != nil {
		fmt.Printf("Nats send message error: %s\n", err.Error())
		return err
	}

	_ = meta.Nats.Flush()
	return nil
}

func PushWithdrawNotify(format, username, amount string) error {

	msg := fmt.Sprintf(format, username, amount, username, amount, username, amount)
	msg = strings.TrimSpace(msg)
	err := meta.Nats.Publish(meta.Prefix+":merchant_notify", []byte(msg))
	fmt.Printf("Nats send a message: %s\n", msg)
	if err != nil {
		fmt.Printf("Nats send message error: %s\n", err.Error())
		return err
	}

	_ = meta.Nats.Flush()
	return nil
}

/*
func SystemLogWrite(content string, ctx *fasthttp.RequestCtx) {

	admin, err := AdminToken(ctx)
	if err != nil {
		fmt.Println("admin not found")
		return
	}

	var privTree PrivTree
	path := string(ctx.Path())
	err = meta.MerchantRedis.HGet(ctx, "priv_tree", path).Scan(&privTree)
	if err != nil {
		fmt.Println("system log get priv_tree", err.Error())
		return
	}

	log := systemLog{
		Title:     privTree.Parent.Parent.Name + "-" + privTree.Parent.Name,
		UID:       admin["id"],
		Name:      admin["name"],
		Content:   content,
		IP:        helper.FromRequest(ctx),
		CreatedAt: ctx.Time().Unix(),
	}

	if err = meta.Zlog.Post(esPrefixIndex("system_log"), log); err != nil {
		fmt.Println("zLog post err: ", err.Error())
	}
}
*/

func TimeFormat(t int64) string {
	return time.Unix(t, 0).In(loc).Format("2006-01-02 15:04:05")
}

func esPrefixIndex(index string) string {
	return meta.EsPrefix + index
}

func Lock(id string) error {

	val := fmt.Sprintf("%s:%s%s", meta.Prefix, defaultRedisKeyPrefix, id)
	ok, err := meta.MerchantRedis.SetNX(ctx, val, "1", LockTimeout).Result()
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}
	if !ok {
		return errors.New(helper.RequestBusy)
	}

	return nil
}

func Unlock(id string) {

	val := fmt.Sprintf("%s:%s%s", meta.Prefix, defaultRedisKeyPrefix, id)
	res, err := meta.MerchantRedis.Unlink(ctx, val).Result()
	if err != nil || res != 1 {
		fmt.Println("Unlock res = ", res)
		fmt.Println("Unlock err = ", err)
	}
}

func paramEncode(args map[string]string) string {

	if len(args) < 1 {
		return ""
	}

	data := url.Values{}
	for k, v := range args {
		data.Set(k, v)
	}
	return data.Encode()
}
