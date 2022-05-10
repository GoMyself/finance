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

// 帐变类型
const (
	TransactionIn                    = 151 //场馆转入
	TransactionOut                   = 152 //场馆转出
	TransactionInFail                = 153 //场馆转入失败补回
	TransactionOutFail               = 154 //场馆转出失败扣除
	TransactionDeposit               = 155 //存款
	TransactionWithDraw              = 156 //提现
	TransactionUpPoint               = 157 //后台上分
	TransactionDownPoint             = 158 //后台下分
	TransactionDownPointBack         = 159 //后台下分回退
	TransactionDividend              = 160 //中心钱包红利派发
	TransactionRebate                = 161 //会员返水
	TransactionFinanceDownPoint      = 162 //财务下分
	TransactionWithDrawFail          = 163 //提现失败
	TransactionValetDeposit          = 164 //代客充值
	TransactionValetWithdraw         = 165 //代客提款
	TransactionAgencyDeposit         = 166 //代理充值
	TransactionAgencyWithdraw        = 167 //代理提款
	TransactionPlatUpPoint           = 168 //后台场馆上分
	TransactionPlatDividend          = 169 //场馆红利派发
	TransactionVIPUpgradeDividend    = 170 //vip升级红利
	TransactionFirstDepositDividend  = 171 //首存活动红利
	TransactionInviteDividend        = 172 //邀请好友红利
	TransactionBet                   = 173 //投注
	TransactionBetCancel             = 174 //投注取消
	TransactionPayout                = 175 //派彩
	TransactionResettlePlus          = 176 //重新结算加币
	TransactionResettleDeduction     = 177 //重新结算减币
	TransactionCancelPayout          = 178 //取消派彩
	TransactionPromoPayout           = 179 //场馆活动派彩
	TransactionEBetTCPrize           = 600 //EBet宝箱奖金
	TransactionEBetLimitRp           = 601 //EBet限量红包
	TransactionEBetLuckyRp           = 602 //EBet幸运红包
	TransactionEBetMasterPayout      = 603 //EBet大赛派彩
	TransactionEBetMasterRegFee      = 604 //EBet大赛报名费
	TransactionEBetBetPrize          = 605 //EBet投注奖励
	TransactionEBetReward            = 606 //EBet打赏
	TransactionEBetMasterPrizeDeduct = 607 //EBet大赛奖金取回
	TransactionWMReward              = 608 //WM打赏
	TransactionSBODividend           = 609 //SBO红利
	TransactionSBOReward             = 610 //SBO打赏
	TransactionSBOBuyLiveCoin        = 611 //SBO 购买LiveCoin
	TransactionSignDividend          = 612 //天天签到活动红利
	TransactionCQ9Dividend           = 613 //CQ9游戏红利
	TransactionCQ9PromoPayout        = 614 //CQ9活动派彩
	TransactionPlayStarPrize         = 615 //Playstar积宝奖金
	TransactionSpadeGamingRp         = 616 //SpadeGaming红包
	TransactionAEReward              = 617 //AE打赏
	TransactionAECancelReward        = 618 //AE取消打赏
	TransactionOfflineDeposit        = 619 //线下转卡存款
	TransactionUSDTOfflineDeposit    = 620 //USDT线下存款
	TransactionEVOPrize              = 621 //游戏奖金(EVO)
	TransactionEVOPromote            = 622 //推广(EVO)
	TransactionEVOJackpot            = 623 //头奖(EVO)
)

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

type systemLog struct {
	Title     string `msg:"title"`
	UID       string `msg:"uid"`
	Name      string `msg:"name"`
	Content   string `msg:"content"`
	IP        string `msg:"ip"`
	CreatedAt int64  `msg:"created_at"`
}

type Priv struct {
	ID        int64  `db:"id" json:"id" redis:"id"`                      //
	Name      string `db:"name" json:"name" redis:"name"`                //权限名字
	Module    string `db:"module" json:"module" redis:"module"`          //模块
	Sortlevel string `db:"sortlevel" json:"sortlevel" redis:"sortlevel"` //
	State     int    `db:"state" json:"state" redis:"state"`             //0:关闭1:开启
	Pid       int64  `db:"pid" json:"pid" redis:"pid"`                   //父级ID
}

type PrivTree struct {
	*Priv
	Parent *PrivTree `json:"parent"`
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
