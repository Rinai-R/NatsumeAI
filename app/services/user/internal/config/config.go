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

	AuthRpc zrpc.RpcClientConf

	Consul consul.Conf

	RedisConf redis.RedisConf
	MysqlConf sqlx.SqlConf
	CacheConf cache.CacheConf

	LogConf logx.LogConf
}

