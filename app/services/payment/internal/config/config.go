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

	Consul consul.Conf

	MysqlConf sqlx.SqlConf
	RedisConf redis.RedisConf
	CacheConf cache.CacheConf

	AsynqConf       AsynqRedisConf
	AsynqServerConf AsynqServerConf

	OrderRpc zrpc.RpcClientConf

	LogConf logx.LogConf

	PaymentTimeoutMinutes int
	SnowflakeNode         int64
}

type AsynqRedisConf struct {
	Addr string
}

type AsynqServerConf struct {
	Concurrency int
	Queues      map[string]int
}
