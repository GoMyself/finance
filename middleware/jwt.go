package middleware

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"

	"finance/contrib/helper"
	"finance/contrib/session"
	"finance/model"
)

var allows = map[string]bool{
	"/finance/callback/vtd":       true,
	"/finance/callback/vtw":       true,
	"/finance/callback/wd":        true,
	"/finance/callback/ww":        true,
	"/finance/callback/uzd":       true,
	"/finance/callback/uzw":       true,
	"/finance/callback/pnw":       true,
	"/finance/callback/pnd":       true,
	"/finance/callback/g7d":       true,
	"/finance/callback/g7w":       true,
	"/finance/callback/fw":        true,
	"/finance/callback/fd":        true,
	"/finance/callback/fconfirm":  true,
	"/finance/version":            true,
	"/finance/pprof/":             true,
	"/finance/pprof/block":        true,
	"/finance/pprof/allocs":       true,
	"/finance/pprof/cmdline":      true,
	"/finance/pprof/goroutine":    true,
	"/finance/pprof/heap":         true,
	"/finance/pprof/profile":      true,
	"/finance/pprof/trace":        true,
	"/finance/pprof/threadcreate": true,
	"/finance/callback/yfbd":      true,
	"/finance/callback/yfbw":      true,
	"/finance/callback/kd":        true,
	"/finance/callback/kw":        true,
	"/finance/callback/fyd":       true,
	"/finance/callback/fyw":       true,
	"/finance/callback/quickd":    true,
	"/finance/callback/quickw":    true,
	"/finance/callback/qxd":       true,
	"/finance/callback/qxw":       true,
	"/finance/callback/usdtd":     true,
	"/finance/callback/xxw":       true,
	"/finance/callback/qqfd":      true,
	"/finance/callback/ynd":       true,
	"/finance/callback/ynw":       true,
	"/finance/callback/vnd":       true,
	"/finance/callback/vnw":       true,
}

// 哪些路由不用动态密码验证
var otpIgnore = map[string]bool{
	"/merchant/finance/vip/list":             true,
	"/merchant/finance/channel/list":         true,
	"/merchant/finance/cate/list":            true,
	"/merchant/finance/promo/detail":         true,
	"/merchant/finance/channel/cache":        true,
	"/merchant/finance/cate/cache":           true,
	"/merchant/finance/deposit/list":         true,
	"/merchant/finance/tunnel/list":          true,
	"/merchant/finance/deposit/history":      true,
	"/merchant/finance/memberlock/list":      true,
	"/merchant/finance/bank/list":            true,
	"/merchant/finance/membercredit/list":    true,
	"/merchant/finance/cate/withdraw":        true,
	"/merchant/finance/withdraw/memberlist":  true,
	"/merchant/finance/withdraw/financelist": true,
	"/merchant/finance/withdraw/historylist": true,
	"/merchant/finance/withdraw/hanguplist":  true,
	"/merchant/finance/credit/list":          true,

	"/merchant/finance/risks/receives":       true,
	"/merchant/finance/risks/state":          true,
	"/merchant/finance/withdraw/waitreview":  true,
	"/merchant/finance/withdraw/riskhistory": true,
	"/merchant/finance/risks/number":         true,
	"/merchant/finance/risks/list":           true,
	"/merchant/finance/withdraw/waitreceive": true,
	"/merchant/finance/withdraw/receive":     true,
	"/merchant/finance/withdraw/cardrecord":  true,
}

func CheckTokenMiddleware(ctx *fasthttp.RequestCtx) error {

	path := string(ctx.Path())

	if _, ok := allows[path]; ok {
		return nil
	}

	data, err := session.Get(ctx)
	if err != nil {
		// fmt.Printf("%s get token from ctx failed:%s\n",path, err.Error())
		fmt.Println("err = ", err)
		return errors.New(`{"status":false,"data":"token"}`)
	}

	has := strings.HasPrefix(path, "/merchant/")
	_, ok := otpIgnore[path]
	if has && !ok && !otp(ctx, data) {
		// return errors.New(`{"status":false,"data":"otp"}`)
	}

	if has {

		gid := fastjson.GetString(data, "group_id")
		fmt.Println("path = ", path)
		fmt.Println("gid = ", gid)

		permission := model.PrivCheck(path, gid)
		fmt.Println("permission = ", permission)

		if permission != nil {
			return errors.New(`{"status":false,"data":"permission denied"}`)
		}
	}

	ctx.SetUserValue("token", data)

	return nil
}

func otp(ctx *fasthttp.RequestCtx, data []byte) bool {

	seamo := ""
	if ctx.IsPost() {
		seamo = string(ctx.PostArgs().Peek("code"))
	} else if ctx.IsGet() {
		seamo = string(ctx.QueryArgs().Peek("code"))
	} else {
		return false
	}

	key := fastjson.GetString(data, "seamo")

	// fmt.Println("seamo= ", seamo)
	// fmt.Println("key= ", key)
	slat := helper.TOTP(key, 15)
	if s, err := strconv.Atoi(seamo); err != nil || s != slat {
		return false
	}
	// fmt.Println("seamo = ", key)

	return true
}
