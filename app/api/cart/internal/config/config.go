// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
	"github.com/zeromicro/zero-contrib/zrpc/registry/consul"
)

type Config struct {
	rest.RestConf

	AuthRpc zrpc.RpcClientConf
	CartRpc zrpc.RpcClientConf

	Consul consul.Conf

	LogConf logx.LogConf
}
