package model

import (
	"finance/contrib/helper"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
	"time"
)

type Config struct {
	ID      int64  `db:"id" json:"id"`
	Name    string `db:"name" json:"name"`
	Content string `db:"content" json:"content"`
}

// ConfigSet 设置配置
func ConfigSet(config *Config) error {

	q := fmt.Sprintf(
		`insert into f_config(name, content,prefix) values ('%s', '%s','%s') on duplicate key update name = '%s', content = '%s',prefix= '%s'`,
		config.Name, config.Content, meta.Prefix, config.Name, config.Content, meta.Prefix,
	)
	_, err := meta.MerchantDB.Exec(q)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return configRedisUpdate(config)
}

// ConfigToRedis 加载配置到redis
func ConfigToRedis() error {

	var cs []Config
	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	ex := g.Ex{
		"prefix": meta.Prefix,
	}
	q, _, _ := dialect.From("f_config").Select("*").Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&cs, q)
	if err != nil {
		fmt.Println("ConfigToRedis Select = ", err)
		return err
	}

	for _, val := range cs {
		pipe.Unlink(ctx, val.Name)
		pipe.Set(ctx, val.Name, val.Content, time.Duration(100)*time.Hour)
		pipe.Persist(ctx, val.Name)
	}

	_, err = pipe.Exec(ctx)
	return err
}

// configRedisUpdate 更新配置缓存
func configRedisUpdate(config *Config) error {

	pipe := meta.MerchantRedis.TxPipeline()
	defer pipe.Close()

	pipe.Set(ctx, config.Name, config.Content, time.Duration(100)*time.Hour)
	pipe.Persist(ctx, config.Name)
	_, err := pipe.Exec(ctx)

	return err
}
