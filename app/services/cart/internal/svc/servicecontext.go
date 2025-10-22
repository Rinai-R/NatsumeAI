package svc

import (
	"NatsumeAI/app/dal/cart"
	"NatsumeAI/app/services/cart/internal/config"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config config.Config
	CartModel cart.CartModel

	Redis *redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {
	redisClient, err := redis.NewRedis(c.RedisConf)
	if err != nil {
		panic("fail to init redis")
	}
	return &ServiceContext{
		Config: c,
		CartModel:  cart.NewCartModel(sqlx.MustNewConn(c.MysqlConf), c.CacheConf),
		Redis: redisClient,
	}
}
