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
	AgentRpc zrpc.RpcClientConf

	Consul consul.Conf

	RedisConf redis.RedisConf
	MysqlConf sqlx.SqlConf
	CacheConf cache.CacheConf

    KafkaConf KafkaConf

    LogConf logx.LogConf

    // Optional: DTM configuration to use commit-and-submit pattern
    DtmConf DtmConf
}


type KafkaConf struct {
    Broker       []string
    Group        string
    MerchantReviewTopic string
}

type DtmConf struct {
    Server     string
    GrpcServer string
    BusiURL    string
    BusiListen string
}
