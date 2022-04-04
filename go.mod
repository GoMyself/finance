module finance

go 1.16

replace github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.5

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

require (
	bitbucket.org/nwf2013/schema v0.0.0-20210517071446-174e527a5076
	github.com/beanstalkd/go-beanstalk v0.1.0
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869
	github.com/coreos/bbolt v0.0.0-00010101000000-000000000000 // indirect
	github.com/coreos/etcd v3.3.27+incompatible
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/doug-martin/goqu/v9 v9.12.0
	github.com/eclipse/paho.mqtt.golang v1.3.5
	github.com/fasthttp/router v1.4.7
	github.com/fluent/fluent-logger-golang v1.6.0
	github.com/go-redis/redis/v8 v8.8.2
	github.com/go-sql-driver/mysql v1.6.0
	github.com/goccy/go-json v0.9.6
	github.com/google/btree v1.0.1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/ipipdotnet/ipdb-go v1.3.1
	github.com/jmoiron/sqlx v1.3.3
	github.com/jonboulle/clockwork v0.2.3 // indirect
	github.com/logrusorgru/aurora v2.0.3+incompatible
	github.com/minio/md5-simd v1.1.2
	github.com/minio/minio-go/v7 v7.0.23
	github.com/modern-go/reflect2 v1.0.2
	github.com/nats-io/nats-server/v2 v2.1.2 // indirect
	github.com/nats-io/nats.go v1.9.1
	github.com/olivere/elastic/v7 v7.0.24
	github.com/panjf2000/ants/v2 v2.4.8
	github.com/pelletier/go-toml v1.9.4
	github.com/prometheus/client_golang v1.12.1 // indirect
	github.com/shopspring/decimal v1.2.0
	github.com/silenceper/pool v1.0.0
	github.com/soheilhy/cmux v0.1.5 // indirect
	github.com/spaolacci/murmur3 v1.1.0
	github.com/tinylib/msgp v1.1.5
	github.com/tmc/grpc-websocket-proxy v0.0.0-20220101234140-673ab2c3ae75 // indirect
	github.com/valyala/fasthttp v1.34.0
	github.com/valyala/fastjson v1.6.3
	github.com/valyala/gorpc v0.0.0-20160519171614-908281bef774
	github.com/wI2L/jettison v0.7.1
	github.com/wenzhenxi/gorsa v0.0.0-20210524035706-528c7050d703
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	github.com/xxtea/xxtea-go v0.0.0-20170828040851-35c4b17eecf6
	go.uber.org/automaxprocs v1.4.0
	go.uber.org/zap v1.21.0 // indirect
	golang.org/x/time v0.0.0-20220224211638-0e9765cccd65 // indirect
	lukechampine.com/frand v1.4.2
	sigs.k8s.io/yaml v1.3.0 // indirect
)
