package controller

import (
	"finance/contrib/helper"
	"finance/model"
	"fmt"

	"github.com/valyala/fasthttp"
)

type PayController struct{}

var newestPay = map[string]bool{

	"175967249852971781": true, // QuickPay momo
	"175967249867358245": true, // QuickPay zalo
	"175967249867679749": true, // QuickPay noline
	"175967249868007831": true, // QuickPay unionpay

	"788985881330068959": true, // 凤扬 momo
	"788985881384976802": true, // 凤扬 zalo
	"788985881388424945": true, // 凤扬 noline
	"788985881400323109": true, // 凤扬 unionpay

	"294920376327426417": true, // uzPay momo
	"573670253791278828": true, // uzPay zolo
	"728685274876253873": true, // uzPay online
	"749883428925626597": true, // uzPay remit

	"100639645688464773": true, //w pay remit
	"100642286454224324": true, //w pay unionpay
	"100632927094061690": true, //w pay momo
	"100636907654161972": true, //w pay online

	"674210725602913159": true, //yfbPay momo
	"674244404001786797": true, //yfbPay zalo
	"677321392621630478": true, //yfbPay online
	"677347106000800957": true, //yfbPay remit
	"677357877705709949": true, //yfbPay unionpay
	"674274075637709353": true, //yfbPay viettelpay

	"349773156100039250": true, // 越南支付 复制转卡
	"308776750008524358": true, // 越南支付 扫码转卡 unionpay

	"136705506399541635": true, // vt支付 momo
	"136886346597697863": true, // vt支付 zalo
	"136895233680932862": true, // vt支付 Viettelpay
	"136918980872302649": true, // vt支付 Online

	"153916130564010419": true, // 918支付 momo
	"153925975222998451": true, // 918支付 online
	"153934719126025455": true, // 918支付 zalo
	"153950213488642272": true, // 918支付 Viettelpay
	"153985081880918463": true, //918支付 Chuyển khoản 转卡

	"171560943702910226": true, // VN支付 Online
	"439141987451271871": true, // VN支付 Offline
	"440046584965688018": true, // VN支付 MOMO
	"440058675832531078": true, // VN支付 QR Banking

	"212584594583418214": true, // 帝宝支付 momo
	"212601768213089447": true, // 帝宝支付 zalo
	"212609704345171478": true, // 帝宝支付 Online
	"212634467162268477": true, // 帝宝支付 Chuyển khoản
	"228065553055909456": true, // 帝宝支付 viettelpay

}

var coinPay = map[string]bool{
	"101003754213878523": true, // USDT1 usdt 第一家收款渠道
}

func (that *PayController) Pay(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	amount := string(ctx.PostArgs().Peek("amount"))
	bid := string(ctx.PostArgs().Peek("bankid"))

	if !helper.CtypeDigit(amount) {
		helper.Print(ctx, false, helper.AmountErr)
	}

	user, err := model.MemberCache(ctx)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	// 新支付走if里面的代码
	if _, ok := newestPay[id]; ok {
		model.NewestPay(ctx, id, amount, bid, user)
		return
	}

	// usdt支付走if里面的代码
	if _, ok := coinPay[id]; ok {
		model.CoinPay(ctx, id, amount, user)
		return
	}

	helper.PrintJson(ctx, false, "404")
}

func (that *PayController) Tunnel(ctx *fasthttp.RequestCtx) {

	id := string(ctx.QueryArgs().Peek("id"))
	if !helper.CtypeDigit(id) {
		helper.Print(ctx, false, helper.IDErr)
		return
	}

	data, err := model.Tunnel(ctx, id)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.PrintJson(ctx, true, data)
}

func (that *PayController) Cate(ctx *fasthttp.RequestCtx) {

	data, err := model.Cate(ctx)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.PrintJson(ctx, true, data)
}

// Manual 发起线下转卡
func (that *PayController) Manual(ctx *fasthttp.RequestCtx) {

	id := string(ctx.PostArgs().Peek("id"))
	amount := string(ctx.PostArgs().Peek("amount"))
	bid := string(ctx.PostArgs().Peek("bankcard_id"))
	bankCode := string(ctx.PostArgs().Peek("bank_code"))
	fmt.Println("id:", id)
	fmt.Println("Manual: ", string(ctx.PostBody()))

	if id != "767158011957916898" {
		helper.Print(ctx, false, helper.ChannelIDErr)
		return
	}

	if !helper.CtypeDigit(amount) {
		helper.Print(ctx, false, helper.AmountErr)
		return
	}

	if bid == "" || bankCode == "" {
		helper.Print(ctx, false, helper.BankcardIDErr)
		return
	}

	res, err := model.Manual(ctx, id, amount, bid, bankCode)
	if err != nil {
		helper.Print(ctx, false, err.Error())
		return
	}

	helper.Print(ctx, false, res)
}
