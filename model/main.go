package model

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"finance/contrib/tracerr"
	"fmt"
	"strings"
	"time"

	"github.com/beanstalkd/go-beanstalk"
	jsoniter "github.com/json-iterator/go"
	"github.com/nats-io/nats.go"

	"bitbucket.org/nwf2013/schema"
	g "github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/gertd/go-pluralize"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/olivere/elastic/v7"
	"github.com/shopspring/decimal"
	cpool "github.com/silenceper/pool"
	"github.com/spaolacci/murmur3"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"github.com/valyala/fastjson"
	"github.com/valyala/gorpc"
)

type log_t struct {
	ID      string `json:"id" msg:"id"`
	Project string `json:"project" msg:"project"`
	Flags   string `json:"flags" msg:"flags"`
	Fn      string `json:"fn" msg:"fn"`
	File    string `json:"file" msg:"file"`
	Content string `json:"content" msg:"content"`
}

type MetaTable struct {
	Zlog          *fluent.Fluent
	MerchantDB    *sqlx.DB
	MerchantRedis *redis.Client
	ES            *elastic.Client
	Grpc          *gorpc.DispatcherClient
	MQPool        cpool.Pool
	Nats          *nats.Conn
	Prefix        string
	Lang          string
	Fcallback     string
	IsDev         bool
	EsPrefix      string
	Finance       map[string]map[string]interface{}
}

var (
	meta            *MetaTable
	loc             *time.Location
	fc              *fasthttp.Client
	ctx             = context.Background()
	pluralizeClient = pluralize.NewClient()

	dialect              = g.Dialect("mysql")
	zero                 = decimal.NewFromInt(0)
	colTunnel            = helper.EnumFields(Tunnel_t{})
	colCate              = helper.EnumFields(Category{})
	colPayment           = helper.EnumFields(Payment_t{})
	colVip               = helper.EnumFields(Vip{})
	colWithdraw          = helper.EnumFields(Withdraw{})
	colChannelBank       = helper.EnumFields(ChannelBanks{})
	colsDeposit          = helper.EnumFields(Deposit{})
	colCreditLevel       = helper.EnumFields(CreditLevel{})
	colMemberCreditLevel = helper.EnumFields(MemberCreditLevel{})
	colMemberLock        = helper.EnumFields(MemberLock{})
	colBankCard          = helper.EnumFields(BankCard{})
	colsWithdraw         = helper.EnumFields(Withdraw{})
	colsMember           = helper.EnumFields(Member{})
	colsMemberBankcard   = helper.EnumFields(MemberBankCard{})
	closChannelDevice    = helper.EnumFields(ChannelDevice{})

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

	cjson = jsoniter.ConfigCompatibleWithStandardLibrary
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

func Constructor(mt *MetaTable, socks5 string, c *gorpc.Client) {

	meta = mt

	if meta.Lang == "cn" {
		loc, _ = time.LoadLocation("Asia/Shanghai")
	} else if meta.Lang == "vn" || meta.Lang == "th" {
		loc, _ = time.LoadLocation("Asia/Bangkok")
	}

	channelToRedis()
	cateToRedis()

	gorpc.RegisterType([]schema.Enc_t{})
	gorpc.RegisterType([]schema.Dec_t{})

	d := gorpc.NewDispatcher()
	d.AddFunc("Encrypt", func(data []schema.Enc_t) []byte { return nil })
	d.AddFunc("Decrypt", func(data []schema.Dec_t) []byte { return nil })

	meta.Grpc = d.NewFuncClient(c)

	fc = &fasthttp.Client{
		MaxConnsPerHost: 60000,
		TLSConfig:       &tls.Config{InsecureSkipVerify: true},
		ReadTimeout:     time.Second * 10,
		WriteTimeout:    time.Second * 10,
	}

	if socks5 != "0.0.0.0" {
		fc.Dial = fasthttpproxy.FasthttpSocksDialer(socks5)
	}

	NewPayment()

	_, _ = meta.Nats.Subscribe(meta.Prefix+":merchant_notify", func(m *nats.Msg) {
		fmt.Printf("Nats received a message: %s\n", string(m.Data))
	})
}

func pushLog(err error, code string) error {

	err = tracerr.Wrap(err)
	fields := map[string]string{
		"filename": tracerr.SprintSource(err, 2, 2),
		"content":  err.Error(),
		"fn":       code,
		"id":       helper.GenId(),
		"project":  "finance_error",
	}

	fmt.Println(err.Error())
	fmt.Println(tracerr.SprintSource(err, 2, 2))

	l := log_t{
		ID:      helper.GenId(),
		Project: "finance",
		Flags:   code,
		Fn:      "",
		File:    tracerr.SprintSource(err, 2, 2),
		Content: err.Error(),
	}
	//err = tdlog.Info(fields)
	//if err != nil {
	//	fmt.Printf("write tdlog[%#v] err : %s", fields, err.Error())
	//}

	_ = meta.Zlog.Post(esPrefixIndex("finance_error"), l)

	switch code {
	case helper.DBErr, helper.RedisErr, helper.ESErr:
		code = helper.ServerErr
	}

	note := fmt.Sprintf("Hệ thống lỗi %s", fields["id"])
	return errors.New(note)
}

func Close() {
	_ = meta.MerchantDB.Close()
	_ = meta.MerchantRedis.Close()
	meta.MQPool.Release()
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

func (c *PrivTree) UnmarshalBinary(d []byte) error {
	return helper.JsonUnmarshal(d, c)
}

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

func TimeFormat(t int64) string {
	return time.Unix(t, 0).In(loc).Format("2006-01-02 15:04:05")
}

func esPrefixIndex(index string) string {
	return meta.EsPrefix + index
}

func BeanPut(name string, param map[string]interface{}, delay int) (string, error) {

	m := &fasthttp.Args{}
	for k, v := range param {
		if _, ok := v.(string); ok {
			m.Set(k, v.(string))
		}
	}

	c, err := meta.MQPool.Get()
	if err != nil {
		return "sys", err
	}

	if conn, ok := c.(*beanstalk.Conn); ok {

		tube := &beanstalk.Tube{Conn: conn, Name: name}
		_, err = tube.Put(m.QueryString(), 1, time.Duration(delay)*time.Second, 10*time.Minute)
		if err != nil {
			meta.MQPool.Put(c)
			return "sys", err
		}
	}

	//将连接放回连接池中
	return "", meta.MQPool.Put(c)
}

func Lock(id string) error {

	val := fmt.Sprintf("%s%s", defaultRedisKeyPrefix, id)
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

	val := fmt.Sprintf("%s%s", defaultRedisKeyPrefix, id)
	res, err := meta.MerchantRedis.Unlink(ctx, val).Result()
	if err != nil || res != 1 {
		fmt.Println("Unlock res = ", res)
		fmt.Println("Unlock err = ", err)
	}
}
