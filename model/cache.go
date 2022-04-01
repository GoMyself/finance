package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"
	"time"

	g "github.com/doug-martin/goqu/v9"
	"github.com/jmoiron/sqlx"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
)

/*
https://www.cnblogs.com/aaronthon/p/11116160.html
*/

type Bank_t struct {
	BankID    string `db:"bank_id" json:"bank_id"`       //银行ID
	CateID    string `db:"cate_id" json:"cate_id"`       //三方渠道类型ID
	PaymentID string `db:"payment_id" json:"payment_id"` //通道ID
	Code      string `db:"code" json:"code"`             //别名
	ID        string `db:"id" json:"id"`                 //
	Name      string `db:"name" json:"name"`             //银行名称
	Sort      int    `db:"sort" json:"sort"`             //排序
	State     string `db:"state" json:"state"`           //0:关闭1:开启
}

type Tunnel_t struct {
	ID         string `db:"id" json:"id"`                    //
	Name       string `db:"name" json:"name"`                //
	Sort       int    `db:"sort" json:"sort"`                //排序
	PromoState string `db:"promo_state"  json:"promo_state"` //存款优化开关
	Content    string `db:"content"  json:"content"`         //存款优化开关
	//Discount string `db:"discount" json:"discount"` // 存款优惠比例
}

type vip_t struct {
	CateID    string `db:"cate_id"`    //渠道ID
	Comment   string `db:"comment"`    //备注
	Flags     string `db:"flags"`      //1:充值类型2:提现类型
	Fmax      string `db:"fmax"`       //最大金额
	Fmin      string `db:"fmin"`       //最小金额
	ID        string `db:"id"`         //
	PaymentID string `db:"payment_id"` //通道ID
	Vip       string `db:"vip"`        //VIP等级
	State     string `db:"state"`      //0:关闭1:开启
}

type PaymentDetail struct {
	Alias string `db:"alias"` //别名
	ID    int64  `db:"id"`    //
	Name  string `db:"name"`  //银行名称
	Sort  int    `db:"sort"`  //排序
}

type Payment_t struct {
	CateID     string `db:"cate_id" redis:"cate_id" json:"cate_id"`             //渠道ID
	ChannelID  string `db:"channel_id" redis:"channel_id" json:"channel_id"`    //通道id
	Comment    string `db:"comment" redis:"comment" json:"comment"`             //
	CreatedAt  string `db:"created_at" redis:"created_at" json:"created_at"`    //创建时间
	Et         string `db:"et" redis:"et" json:"et"`                            //结束时间
	Fmax       string `db:"fmax" redis:"fmax" json:"fmax"`                      //最大支付金额
	Fmin       string `db:"fmin" redis:"fmin" json:"fmin"`                      //最小支付金额
	Gateway    string `db:"gateway" redis:"gateway" json:"gateway"`             //支付网关
	ID         string `db:"id" redis:"id" json:"id"`                            //
	Quota      string `db:"quota" redis:"quota" json:"quota"`                   //每天限额
	Amount     string `db:"amount" redis:"amount" json:"amount"`                //每天限额
	Sort       string `db:"sort" redis:"sort" json:"sort"`                      //
	St         string `db:"st" redis:"st" json:"st"`                            //开始时间
	State      string `db:"state" redis:"state" json:"state"`                   //0:关闭1:开启
	Devices    string `db:"devices" redis:"devices" json:"devices"`             //设备号
	AmountList string `db:"amount_list" redis:"amount_list" json:"amount_list"` // 固定金额列表
}

func CacheRefreshPaymentBanks(id string) error {

	var banks []Bank_t

	ex := g.Ex{
		"payment_id": id,
		"state":      "1",
		"prefix":     meta.Prefix,
	}
	query, _, _ := dialect.From("f_channel_banks").Select(colChannelBank...).Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&banks, query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	var bankResult []Bank_t
	for _, bank := range banks {

		if bank.PaymentID == "304314961990368154" { // 线下转卡

			num := 0
			ex := g.Ex{
				"channel_bank_id": bank.ID,
				"state":           1,
				"prefix":          meta.Prefix,
			}

			query, _, _ := dialect.From("f_bankcards").Select(g.COUNT("id")).Where(ex).ToSQL()
			err := meta.MerchantDB.Get(&num, query)
			if err != nil {
				return pushLog(err, helper.DBErr)
			}

			if num == 0 {
				continue
			}
		}

		bankResult = append(bankResult, bank)
	}

	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	pipe.Unlink(ctx, "BK:"+id)
	if len(bankResult) > 0 {
		s, err := helper.JsonMarshal(bankResult)
		if err != nil {
			return errors.New(helper.FormatErr)
		}

		pipe.Set(ctx, "BK:"+id, string(s), 999999*time.Hour)
		pipe.Persist(ctx, "BK:"+id)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}

	return nil
}

func CacheRefreshPayment(id string) error {

	val, err := ChanByID(id)
	if err != nil {
		return err
	}

	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	value := map[string]interface{}{
		"amount":      val.Amount,
		"devices":     val.Devices,
		"cate_id":     val.CateID,
		"channel_id":  val.ChannelID,
		"comment":     val.Comment,
		"created_at":  val.CreatedAt,
		"et":          val.Et,
		"fmax":        val.Fmax,
		"fmin":        val.Fmin,
		"gateway":     val.Gateway,
		"id":          val.ID,
		"quota":       val.Quota,
		"sort":        val.Sort,
		"st":          val.St,
		"state":       val.State,
		"amount_list": val.AmountList,
	}
	pipe.Unlink(ctx, "p:"+id)
	pipe.HMSet(ctx, "p:"+id, value)
	pipe.Persist(ctx, "p:"+id)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return pushLog(err, helper.RedisErr)
	}

	return nil
}

// CachePayment 获取支付方式
func CachePayment(id string) (FPay, error) {

	m := FPay{}
	var cols []string

	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	for _, val := range colPayment {
		cols = append(cols, val.(string))
	}

	// 需要执行的命令
	exists := pipe.Exists(ctx, "p:"+id)
	rs := pipe.HMGet(ctx, "p:"+id, cols...)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return m, err
	}

	if exists.Val() == 0 {
		return m, errors.New(helper.RedisErr)
	}

	err = rs.Scan(&m)
	if err != nil {
		return m, err
	}

	return m, nil
}

func Tunnel(fctx *fasthttp.RequestCtx, id string) (string, error) {

	m := Payment_t{}

	u, err := MemberCache(fctx)
	if err != nil {
		return "", err
	}
	key := fmt.Sprintf("p:%d:%s", u.Level, id)
	//sip := helper.FromRequest(fctx)
	//if strings.Count(sip, ":") >= 2 {
	//	key = fmt.Sprintf("p:%d:%s", 9, id)
	//}

	paymentId, err := meta.MerchantRedis.RPopLPush(ctx, key, key).Result()
	if err != nil {
		fmt.Println("SMembers = ", err.Error())
		return "[]", nil
	}

	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	rs := pipe.HMGet(ctx, "p:"+paymentId, "id", "fmin", "fmax", "et", "st", "amount_list")
	re := pipe.HMGet(ctx, "pr:"+paymentId, "fmin", "fmax")
	bk := pipe.Get(ctx, "BK:"+paymentId)

	_, _ = pipe.Exec(ctx)

	if rs.Err() != nil {
		return "", pushLog(err, helper.RedisErr)
	}
	if err := rs.Scan(&m); err != nil {
		return "", pushLog(err, helper.RedisErr)
	}

	var (
		fmin, fmax string
		ok         bool
	)
	scope := re.Val()
	if fmin, ok = scope[0].(string); !ok {
		return "", errors.New(helper.TunnelMinLimitErr)
	}

	if fmax, ok = scope[1].(string); !ok {
		return "", errors.New(helper.TunnelMaxLimitErr)
	}

	base := fastjson.MustParse(`{"id":"0","bank":[], "fmin":"0","fmax":"0", "amount_list": ""}`)
	base.Set("id", fastjson.MustParse(fmt.Sprintf(`"%s"`, m.ID)))
	base.Set("fmin", fastjson.MustParse(fmt.Sprintf(`"%s"`, fmin)))
	base.Set("fmax", fastjson.MustParse(fmt.Sprintf(`"%s"`, fmax)))
	base.Set("amount_list", fastjson.MustParse(fmt.Sprintf(`"%s"`, m.AmountList)))

	banks := bk.Val()
	if len(banks) > 0 {
		base.Set("bank", fastjson.MustParse(banks))
	}

	return base.String(), nil
}

func Cate(fctx *fasthttp.RequestCtx) (string, error) {

	str := CateListRedis()
	return str, nil
}

// CreateAutomatic 创建代付的轮询队列
func CreateAutomatic(level string) {

	var vips []vip_t
	ex := g.Ex{
		"vip":    level,
		"flags":  2,
		"state":  "1",
		"prefix": meta.Prefix,
	}
	query, _, _ := dialect.From("f_vip").Select(colVip...).Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&vips, query)
	if err != nil {
		fmt.Println("1", err.Error())
		return
	}

	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	pipe.Unlink(ctx, "pw:"+level)

	for _, val := range vips {
		value, _ := helper.JsonMarshal(val)
		pipe.LPush(ctx, "pw:"+level, value)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		fmt.Println("err = ", err)
	}
}

func Create(level string) {

	var (
		cIds       []string
		paymentIds []string
		vips       []vip_t
		tunnels    []Tunnel_t
		payments   []Payment_t
	)

	//删除key
	meta.MerchantRedis.Unlink(ctx, "p:"+level).Result()
	ex := g.Ex{
		"vip":    level,
		"flags":  1,
		"state":  "1",
		"prefix": meta.Prefix,
	}
	query, _, _ := dialect.From("f_vip").Select(colVip...).Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&vips, query)
	if err != nil {
		fmt.Println("1", err.Error())
		return
	}

	for _, val := range vips {
		paymentIds = append(paymentIds, val.PaymentID)
	}

	if len(paymentIds) == 0 {
		return
	}

	ex = g.Ex{
		"id":     paymentIds,
		"state":  "1",
		"prefix": meta.Prefix,
	}
	query, _, _ = dialect.From("f_payment").Select(colPayment...).Where(ex).ToSQL()
	queryIn, args, err := sqlx.In(query)
	if err != nil {
		fmt.Println("2", err.Error())
		return
	}

	err = meta.MerchantDB.Select(&payments, queryIn, args...)
	if err != nil {
		fmt.Println("3", err.Error())
		return
	}

	for _, val := range payments {
		cIds = append(cIds, val.ChannelID)
	}

	ex = g.Ex{
		"id":     cIds,
		"prefix": meta.Prefix,
	}
	query, _, _ = dialect.From("f_channel_type").Select(colTunnel...).Where(ex).ToSQL()
	queryIn, args, err = sqlx.In(query)
	if err != nil {
		fmt.Println("4", err.Error())
		return
	}

	err = meta.MerchantDB.Select(&tunnels, queryIn, args...)
	if err != nil {
		fmt.Println("5", err.Error())
		return
	}

	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	for _, val := range payments {
		pipe.Unlink(ctx, "p:"+val.ID)
		pipe.Unlink(ctx, "p:"+level+":"+val.ChannelID)
	}

	for _, val := range vips {
		value := map[string]interface{}{
			"fmax": val.Fmax,
			"fmin": val.Fmin,
		}

		pipe.Unlink(ctx, "pr:"+val.PaymentID)
		pipe.HMSet(ctx, "pr:"+val.PaymentID, value)
		pipe.Persist(ctx, "pr:"+val.PaymentID)
	}

	for _, val := range tunnels {

		value, _ := helper.JsonMarshal(val)
		pipe.SAdd(ctx, "p:"+level, value)
	}

	pipe.Persist(ctx, "p:"+level)

	for _, val := range payments {

		value := map[string]interface{}{
			"amount":      val.Amount,
			"devices":     val.Devices,
			"cate_id":     val.CateID,
			"channel_id":  val.ChannelID,
			"comment":     val.Comment,
			"created_at":  val.CreatedAt,
			"et":          val.Et,
			"fmax":        val.Fmax,
			"fmin":        val.Fmin,
			"gateway":     val.Gateway,
			"id":          val.ID,
			"quota":       val.Quota,
			"sort":        val.Sort,
			"st":          val.St,
			"state":       val.State,
			"amount_list": val.AmountList,
		}
		pipe.LPush(ctx, "p:"+level+":"+val.ChannelID, val.ID)
		pipe.HMSet(ctx, "p:"+val.ID, value)
		pipe.Persist(ctx, "p:"+val.ID)
	}

	_, err = pipe.Exec(ctx)

	fmt.Println("err = ", err)
	fmt.Println("vip = ", vips)
	fmt.Println("tunnels = ", tunnels)
	fmt.Println("payments = ", payments)
	fmt.Println("paymentIds = ", paymentIds)
}
