package config

import (
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
	"github.com/zeromicro/zero-contrib/zrpc/registry/consul"
)

type Config struct {
    zrpc.RpcServerConf

    InventoryRpc zrpc.RpcClientConf
    CouponRpc    zrpc.RpcClientConf
    ProductRpc   zrpc.RpcClientConf

    Consul consul.Conf

    RedisConf redis.RedisConf
    MysqlConf sqlx.SqlConf
    CacheConf cache.CacheConf

    // Use lightweight config structs to avoid mapstructure errors on func fields
    AsynqConf       AsynqRedisConf
    AsynqServerConf AsynqServerConf

    LogConf logx.LogConf

    KafkaConf KafkaConf

    // Preorder expiration in minutes (used to compute ExpireAt and delay tasks)
    PreorderTTLMinutes int
}

// Minimal redis client config for Asynq
type AsynqRedisConf struct {
    Addr     string
}

// Minimal asynq server config
type AsynqServerConf struct {
    Concurrency int
    Queues      map[string]int
}

type KafkaConf struct {
    Broker       []string
    Group        string
    PreOrderTopic string
    OrderTopic    string
}
