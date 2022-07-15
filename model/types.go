package model

import (
	"time"
)

const (
	DepositFlagThird     = 1
	DepositFlagThirdUSTD = 2
	DepositFlagManual    = 3
	DepositFlagUSDT      = 4
)

// 存款状态
const (
	DepositConfirming = 361 //确认中
	DepositSuccess    = 362 //存款成功
	DepositCancelled  = 363 //存款已取消
	DepositReviewing  = 364 //存款审核中
)

// 取款状态
const (
	WithdrawReviewing     = 371 //审核中
	WithdrawReviewReject  = 372 //审核拒绝
	WithdrawDealing       = 373 //出款中
	WithdrawSuccess       = 374 //提款成功
	WithdrawFailed        = 375 //出款失败
	WithdrawAbnormal      = 376 //异常订单
	WithdrawAutoPayFailed = 377 // 代付失败
	WithdrawHangup        = 378 // 挂起
	WithdrawDispatched    = 379 // 已派单
)

// 后台上下分审核状态
const (
	AdjustReviewing    = 256 //后台调整审核中
	AdjustReviewPass   = 257 //后台调整审核通过
	AdjustReviewReject = 258 //后台调整审核不通过
)

// 后台上下分状态
const (
	AdjustFailed      = 261 //上下分失败
	AdjustSuccess     = 262 //上下分成功
	AdjustPlatDealing = 263 //上分场馆处理中
)

// 后台调整类型
const (
	AdjustUpMode   = 251 // 上分
	AdjustDownMode = 252 // 下分
)

const (
	defaultRedisKeyPrefix = "rlock:"
	LockTimeout           = 20 * time.Second
)

/*
type paymentChannel string

var (
	momo       paymentChannel = "momo"
	zalo       paymentChannel = "zalo"
	online     paymentChannel = "online" // 网上银行 直连
	remit      paymentChannel = "remit"  // 银行卡转帐 卡转卡
	coinpay    paymentChannel = "coinpay"
	viettelpay paymentChannel = "viettelpay"
	withdraw   paymentChannel = "withdraw"
	unionpay   paymentChannel = "unionpay"
	offline    paymentChannel = "offline"
	usdt       paymentChannel = "usdt"
	manual     paymentChannel = "manual"
	alipay     paymentChannel = "alipay"
	wechat     paymentChannel = "wechat"
	auto       paymentChannel = "Chuyển Khoản Nhanh"
)

var (
	channelMomo       = "1"
	channelZalo       = "2"
	channelOnline     = "3"
	channelRemit      = "4"
	channelCoinPay    = "5"
	channelViettelPay = "6"
	channelWithdraw   = "7"
	channelUnionPay   = "8"
	channelOffline    = "9"
	channelUSDT       = "10"
	channelManual     = "11"
	channelAlipay     = "12"
	channelWechat     = "13"
	channelAuto       = "14"
)

var channels = map[string]paymentChannel{
	channelMomo:       momo,
	channelZalo:       zalo,
	channelOnline:     online,
	channelRemit:      remit,
	channelCoinPay:    coinpay,
	channelViettelPay: viettelpay,
	channelWithdraw:   withdraw,
	channelUnionPay:   unionpay,
	channelOffline:    offline,
	channelUSDT:       usdt,
	channelManual:     manual,
	channelAlipay:     alipay,
	channelWechat:     wechat,
	channelAuto:       auto,
}
*/

type systemLog struct {
	Title     string `msg:"title"`
	UID       string `msg:"uid"`
	Name      string `msg:"name"`
	Content   string `msg:"content"`
	IP        string `msg:"ip"`
	CreatedAt int64  `msg:"created_at"`
}

// 发起支付返回结果结构
type payCommRes struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// 虚拟币发起支付返回结果结构
type coinPayCommRes struct {
	ID           string `json:"id"`
	Address      string `json:"address"`
	Amount       string `json:"amount"`
	ProtocolType string `json:"protocol_type"`
}

// 虚拟币发起支付返回结果结构
type offlineCommRes struct {
	ID           string `json:"id"`
	Name         string `db:"name" json:"name"`
	CardNo       string `db:"card_no" json:"card_no"`
	RealName     string `db:"real_name" json:"real_name"`
	BankAddr     string `db:"bank_addr" json:"bank_addr"`
	ManualRemark string `db:"-" json:"manual_remark"`
}

// 订单回调response
type paymentCallbackResp struct {
	OrderID string // 我方订单号
	State   int    // 订单状态
	Amount  string // 订单金额
	Cent    int64  // 数据数值差异倍数
	Sign    string // 签名(g7的签名校验需要)
	Resp    interface{}
}

// paymentDepositResp 存款
type paymentDepositResp struct {
	Addr    string                 // 三方返回的充值地址
	OrderID string                 // 三方的订单号, 如果三方没有返回订单号, 这个值则为入参id(即我方订单号)
	Data    map[string]interface{} // 向三方发起http请求的参数以及response data
	IsForm  string
	useLink int //使用地址跳转或重新发起请求 0：使用链接跳转  1：使用订单号重新发起请求
}

// paymentWithdrawalRsp 取款
type paymentWithdrawalRsp struct {
	OrderID string // 三方的订单号, 如果三方没有返回订单号, 这个值则为入参id(即我方订单号)
}

// FPay f_payment表名
type FPay struct {
	CateID    string `db:"cate_id" redis:"cate_id" json:"cate_id"`          //渠道ID
	ChannelID string `db:"channel_id" redis:"channel_id" json:"channel_id"` //通道id
	Comment   string `db:"comment" redis:"comment" json:"comment"`          //
	CreatedAt string `db:"created_at" redis:"created_at" json:"created_at"` //创建时间
	Et        string `db:"et" redis:"et" json:"et"`                         //结束时间
	Fmax      string `db:"fmax" redis:"fmax" json:"fmax"`                   //最大支付金额
	Fmin      string `db:"fmin" redis:"fmin" json:"fmin"`                   //最小支付金额
	Gateway   string `db:"gateway" redis:"gateway" json:"gateway"`          //支付网关
	ID        string `db:"id" redis:"id" json:"id"`                         //
	Quota     string `db:"quota" redis:"quota" json:"quota"`                //每天限额
	Amount    string `db:"amount" redis:"amount" json:"amount"`             //每天限额
	Sort      string `db:"sort" redis:"sort" json:"sort"`                   //
	St        string `db:"st" redis:"st" json:"st"`                         //开始时间
	State     string `db:"state" redis:"state" json:"state"`                //0:关闭1:开启
	Devices   string `db:"devices" redis:"devices" json:"devices"`          //设备号
}

type WithdrawAutoParam struct {
	OrderID     string    // 订单id
	Amount      string    // 金额
	BankID      string    // 银行id
	BankCode    string    // 银行
	CardNumber  string    // 银行卡号
	CardName    string    // 持卡人姓名
	Ts          time.Time // 时间
	PaymentID   string    // 提现渠道信息
	BankAddress string    // 开户支行
}

type paymentTDLog struct {
	Merchant     string `db:"merchant"`
	Channel      string `db:"channel"`
	Flag         string `db:"flag"`
	RequestURL   string `db:"request_url"`
	RequestBody  string `db:"request_body"`
	ResponseCode int    `db:"response_code"`
	ResponseBody string `db:"response_body"`
	Error        string `db:"error"`
	Lable        string `db:"lable"`
	Level        string `db:"level"`
	OrderID      string `db:"order_id"`
	Username     string `db:"username"`
}

type Message struct {
	ID       string `json:"id"`        //会员站内信id
	MsgID    string `json:"msg_id"`    //站内信id
	Username string `json:"username"`  //会员名
	Title    string `json:"title"`     //标题
	SubTitle string `json:"sub_title"` //标题
	Content  string `json:"content"`   //内容
	IsTop    int    `json:"is_top"`    //0不置顶 1置顶
	IsVip    int    `json:"is_vip"`    //0非vip站内信 1vip站内信
	Ty       int    `json:"ty"`        //1站内消息 2活动消息
	IsRead   int    `json:"is_read"`   //是否已读 0未读 1已读
	SendName string `json:"send_name"` //发送人名
	SendAt   int64  `json:"send_at"`   //发送时间
	Prefix   string `json:"prefix"`    //商户前缀
}

type FirstDeposit struct {
	DepositAt int64  `db:"deposit_at"`
	Uid       string `db:"uid"`
}

type StateNum struct {
	T     int   `json:"t" db:"t"`
	State int64 `json:"state" db:"state"`
}
