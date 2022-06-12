package main

import (
	"context"
	"finance/contrib/apollo"
	"finance/contrib/conn"
	"finance/contrib/helper"
	"finance/contrib/session"
	"finance/middleware"
	"finance/model"
	"finance/router"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/lucacasonato/mqtt"
	"github.com/valyala/fasthttp"
	_ "go.uber.org/automaxprocs"
)

var (
	gitReversion   = ""
	buildTime      = ""
	buildGoVersion = ""
)

func main() {

	var (
		err error
		ctx = context.Background()
	)

	cfg := conf{}
	argc := len(os.Args)
	if argc != 4 {
		fmt.Printf("%s <etcds> <cfgPath> <sock5|load>\r\n", os.Args[0])
		return
	}

	endpoints := strings.Split(os.Args[1], ",")

	apollo.New(endpoints)
	apollo.Parse(os.Args[2], &cfg)
	content, err := apollo.ParseToml(path.Dir(os.Args[2])+"/finance.toml", false)
	apollo.Close()
	if err != nil {
		log.Fatalln(err)
	}

	mt := new(model.MetaTable)

	mt.MerchantTD = conn.InitTD(cfg.Td.Addr, cfg.Td.MaxIdleConn, cfg.Td.MaxOpenConn)
	mt.MerchantDB = conn.InitDB(cfg.Db.Master.Addr, cfg.Db.Master.MaxIdleConn, cfg.Db.Master.MaxOpenConn)
	mt.ES = conn.InitES(cfg.Es.Host, cfg.Es.Username, cfg.Es.Password)
	mt.MerchantRedis = conn.InitRedisCluster(cfg.Redis.Addr, cfg.Redis.Password)

	mt.MerchantNats, err = mqtt.NewClient(mqtt.ClientOptions{
		// required
		Servers: cfg.Nats.Servers,

		// optional
		ClientID:      helper.GenId(),
		Username:      "admin",
		Password:      "***",
		AutoReconnect: true,
	})
	if err != nil {
		panic(err)
	}

	err = mt.MerchantNats.Connect(ctx)
	if err != nil {
		panic(err)
	}

	bin := strings.Split(os.Args[0], "/")
	mt.Program = bin[len(bin)-1]

	mt.Prefix = cfg.Prefix
	mt.EsPrefix = cfg.EsPrefix
	mt.Lang = cfg.Lang
	mt.Fcallback = cfg.Fcallback
	mt.IsDev = cfg.IsDev

	mt.Finance = content
	model.Constructor(mt, os.Args[3], cfg.Rpc)

	session.New(mt.MerchantRedis, mt.Prefix)

	defer func() {
		model.Close()
		mt = nil
	}()

	if os.Args[3] == "load" {
		fmt.Println("load")

		model.TunnelUpdateCache()
		for i := 1; i < 11; i++ {
			level := fmt.Sprintf("%d", i)
			model.Create(level)
		}

		model.TransacCodeCreate()
		model.ChannelTypeCreateCache()
		model.BankCardUpdateCache()
		return
	}

	if os.Args[3] == "cleanCard" {
		fmt.Println("cleanBankFinshAmount")

		model.CleanBankFinshAmount()
		return
	}

	b := router.BuildInfo{
		GitReversion:   gitReversion,
		BuildTime:      buildTime,
		BuildGoVersion: buildGoVersion,
	}
	app := router.SetupRouter(b)
	srv := &fasthttp.Server{
		Handler:            middleware.Use(app.Handler),
		ReadTimeout:        router.ApiTimeout,
		WriteTimeout:       router.ApiTimeout,
		Name:               "finance",
		MaxRequestBodySize: 51 * 1024 * 1024,
	}
	fmt.Printf("gitReversion = %s\r\nbuildGoVersion = %s\r\nbuildTime = %s\r\n", gitReversion, buildGoVersion, buildTime)
	fmt.Println("Finance running", cfg.Port.Finance)
	if err := srv.ListenAndServe(cfg.Port.Finance); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}
