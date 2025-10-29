// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
    commoncfg "NatsumeAI/app/common/config"
    "github.com/zeromicro/go-zero/core/logx"
    "github.com/zeromicro/go-zero/rest"
    "github.com/zeromicro/go-zero/zrpc"
    "github.com/zeromicro/zero-contrib/zrpc/registry/consul"
)

type Config struct {
    rest.RestConf

    Consul consul.Conf

    AuthRpc zrpc.RpcClientConf
    InventoryRpc zrpc.RpcClientConf

    LogConf logx.LogConf

    CasbinMiddleware commoncfg.CasbinMiddlewareConf
}
