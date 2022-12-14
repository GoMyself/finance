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

	"github.com/lucacasonato/mqtt"

	g "github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/go-redis/redis/v8"
	"github.com/hprose/hprose-golang/v3/rpc/core"
	rpchttp "github.com/hprose/hprose-golang/v3/rpc/http"
	. "github.com/hprose/hprose-golang/v3/rpc/http/fasthttp"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/spaolacci/murmur3"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"github.com/valyala/fastjson"
)

type MetaTable struct {
	MerchantDB    *sqlx.DB
	MerchantTD    *sqlx.DB
	MerchantLogTD *sqlx.DB
	MerchantRedis *redis.ClusterClient
	MerchantMqtt  *mqtt.Client
	Program       string
	Prefix        string
	Lang          string
	IndexUrl      string
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
	FinshDepositFlow func(rctx context.Context, username, billNo, adminId, adminName string) bool
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
	colsMemberInfo       = helper.EnumFields(MemberInfo{})
)

var (
	paymentRoute  = map[string]Payment{}
	paymentLogTag = "payment_log"
	// ??????redis?????????????????????key
	depositOrderLockKey = "d:order:%s"
	// ??????redis?????????????????????key
	withdrawOrderLockKey = "w:order:%s"
)

//????????????
const (
	MemberWallet = 1 //??????????????????
	AgencyWallet = 2 //?????????????????????
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

	_ = cateToRedis()

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
		"ts":       ts.In(loc).UnixMicro(),
	}

	query, _, _ := dialect.Insert("finance_error").Rows(&fields).ToSQL()
	fmt.Println(query)
	_, err1 := meta.MerchantLogTD.Exec(query)
	if err1 != nil {
		fmt.Println("insert goerror query = ", query)
		fmt.Println("insert goerror err = ", err1.Error())
	}

	return fmt.Errorf("h??? th???ng l???i %s", id)
}

func tdInsert(tbl string, record g.Record) {

	query, _, _ := dialect.Insert(tbl).Rows(record).ToSQL()
	fmt.Println(query)
	_, err := meta.MerchantTD.Exec(query)
	if err != nil {
		fmt.Println("update td = ", err.Error(), record)
	}
}

func Close() {
	meta.MerchantTD.Close()
	_ = meta.MerchantDB.Close()
	_ = meta.MerchantRedis.Close()
	meta.MerchantMqtt.DisconnectImmediately()
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

// ??????admin???name
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

	topic := fmt.Sprintf("%s/merchant", meta.Prefix)
	err := meta.MerchantMqtt.Publish(ctx, topic, []byte(msg), mqtt.AtLeastOnce)
	if err != nil {
		fmt.Println("merchantNats.Publish finance = ", err.Error())
		return err
	}

	return nil
}

func PushWithdrawNotify(format, username, amount string) error {

	ts := time.Now()
	msg := fmt.Sprintf(format, username, amount, username, amount, username, amount)
	msg = strings.TrimSpace(msg)

	topic := fmt.Sprintf("%s/merchant", meta.Prefix)
	err := meta.MerchantMqtt.Publish(ctx, topic, []byte(msg), mqtt.AtLeastOnce)
	if err != nil {
		fmt.Println("failed", time.Since(ts), err.Error())
		fmt.Println("merchantNats.Publish finance = ", err.Error())
		return err
	}

	fmt.Println("success", time.Since(ts))

	return nil
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
