package config

import (
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"github.com/zeromicro/zero-contrib/zrpc/registry/consul"
)

type Config struct {
	zrpc.RpcServerConf
	Consul consul.Conf

	ChatModel ModelConf
	Embedding ModelConf
	Rerank ModelConf

	LogConf logx.LogConf
}


type ModelConf struct {
	BaseUrl string
	APIKey string
	Model string
}