package svc

import (
	"NatsumeAI/app/dal/inventory"
	"NatsumeAI/app/services/inventory/internal/config"

	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

const (
	preInventoryTokenKey          = "inventory:pre:deduct"
	preInventoryTokenLimiterRate  = 1000
	preInventoryTokenLimiterBurst = 1000
)

type ServiceContext struct {
	Config config.Config

	InventoryModel inventory.InventoryModel

	Redis *redis.Redis

	InventoryPreDeductLimiter *limit.TokenLimiter
}

func NewServiceContext(c config.Config) *ServiceContext {
	inventoryModel := inventory.NewInventoryModel(sqlx.MustNewConn(c.MysqlConf), c.CacheConf)
	redisClient, err := redis.NewRedis(c.RedisConf)
	if err != nil {
		panic("fail to init redis")
	}

	preDeductLimiter := limit.NewTokenLimiter(
		preInventoryTokenLimiterRate,
		preInventoryTokenLimiterBurst,
		redisClient,
		preInventoryTokenKey,
	)

	return &ServiceContext{
		Config:                    c,
		InventoryModel:            inventoryModel,
		Redis:                     redisClient,
		InventoryPreDeductLimiter: preDeductLimiter,
	}
}
