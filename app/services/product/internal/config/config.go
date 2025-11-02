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

	InventoryRpc zrpc.RpcClientConf

	RedisConf redis.RedisConf
	MysqlConf sqlx.SqlConf
	CacheConf cache.CacheConf

	LogConf     logx.LogConf
	ElasticConf ElasticConf
	Embedding   EmbeddingConf
}

type ElasticConf struct {
	Addresses          []string
	Username           string
	Password           string
	IndexName          string
	EmbeddingDimension int
	Alpha              float64
}

type EmbeddingConf struct {
	BaseURL string
	APIKey  string
	Model   string
}
