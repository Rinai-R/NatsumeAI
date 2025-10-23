package svc

import (
	"NatsumeAI/app/dal/inventory"
	"NatsumeAI/app/services/inventory/internal/config"

	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config config.Config

	InventoryModel      inventory.InventoryModel
	InventoryAuditModel inventory.InventoryAuditModel

	InventoryTokenModel inventory.InventoryTokenModel

	InventoryPreDeductLimiter *limit.TokenLimiter
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.MustSetup(c.LogConf)
	inventoryModel := inventory.NewInventoryModel(sqlx.MustNewConn(c.MysqlConf), c.CacheConf)
	inventoryAuditModel := inventory.NewInventoryAuditModel(sqlx.MustNewConn(c.MysqlConf), c.CacheConf)
	redisClient, err := redis.NewRedis(c.RedisConf)
	if err != nil {
		panic("fail to init redis")
	}
	return &ServiceContext{
		Config:              c,
		InventoryModel:      inventoryModel,
		InventoryAuditModel: inventoryAuditModel,
		InventoryTokenModel: inventory.NewInventoryTokenModel(redisClient, inventoryModel),
	}
}
