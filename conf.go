package main

type conf struct {
	Lang         string `json:"lang"`
	Prefix       string `json:"prefix"`
	EsPrefix     string `json:"es_prefix"`
	PullPrefix   string `json:"pull_prefix"`
	IsDev        bool   `json:"is_dev"`
	Sock5        string `json:"sock5"`
	Rpc          string `json:"rpc"`
	IndexUrl     string `json:"index_url"`
	Fcallback    string `json:"fcallback"`
	AutoPayLimit string `json:"autoPayLimit"`
	Nats         struct {
		Servers  []string `json:"servers"`
		Username string   `json:"username"`
		Password string   `json:"password"`
	} `json:"nats"`
	Beanstalkd struct {
		Addr    string `json:"addr"`
		MaxIdle int    `json:"maxIdle"`
		MaxCap  int    `json:"maxCap"`
	} `json:"beanstalkd"`
	Db struct {
		Master struct {
			Addr        string `json:"addr"`
			MaxIdleConn int    `json:"max_idle_conn"`
			MaxOpenConn int    `json:"max_open_conn"`
		} `json:"master"`
		Report struct {
			Addr        string `json:"addr"`
			MaxIdleConn int    `json:"max_idle_conn"`
			MaxOpenConn int    `json:"max_open_conn"`
		} `json:"report"`
		Bet struct {
			Addr        string `json:"addr"`
			MaxIdleConn int    `json:"max_idle_conn"`
			MaxOpenConn int    `json:"max_open_conn"`
		} `json:"bet"`
	} `json:"db"`
	Td struct {
		Log struct {
			Addr        string `json:"addr"`
			MaxIdleConn int    `json:"max_idle_conn"`
			MaxOpenConn int    `json:"max_open_conn"`
		} `json:"log"`
		Message struct {
			Addr        string `json:"addr"`
			MaxIdleConn int    `json:"max_idle_conn"`
			MaxOpenConn int    `json:"max_open_conn"`
		} `json:"message"`
	} `json:"td"`
	BankcardValidAPI struct {
		URL string `json:"url"`
		Key string `json:"key"`
	} `json:"bankcard_valid_api"`
	Redis struct {
		Addr     []string `json:"addr"`
		Password string   `json:"password"`
	} `json:"redis"`
	Minio struct {
		ImagesBucket    string `json:"images_bucket"`
		JsonBucket      string `json:"json_bucket"`
		Endpoint        string `json:"endpoint"`
		AccessKeyID     string `json:"accessKeyID"`
		SecretAccessKey string `json:"secretAccessKey"`
		UseSSL          bool   `json:"useSSL"`
		UploadUrl       string `json:"uploadUrl"`
	} `json:"minio"`
	Es struct {
		Host     []string `json:"host"`
		Username string   `json:"username"`
		Password string   `json:"password"`
	} `json:"es"`
	Port struct {
		Game     string `json:"game"`
		Member   string `json:"member"`
		Promo    string `json:"promo"`
		Merchant string `json:"merchant"`
		Finance  string `json:"finance"`
	} `json:"port"`
}
