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

    // DTM configuration (optional). When configured, checkout uses DTM Msg
    // to atomically commit preorder insert and submit a delivery step that
    // publishes the checkout event (replacing local outbox).
    DtmConf DtmConf
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

// DtmConf configures DTM server and our business callback URL.
type DtmConf struct {
    // Server is the dtm server base, for example: http://dtm:36789/api/dtmsvr
    Server  string
    // BusiURL is the base http url for callbacks, for example: http://order:8180
    // We will register handlers under /dtm/* on this server.
    BusiURL string
    // BusiListen is the local bind address for the callback HTTP server,
    // e.g. ":8180" or "0.0.0.0:8180". This can differ from BusiURL's host
    // because containers may expose different names/ports externally.
    BusiListen string
}
