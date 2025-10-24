// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
    "flag"
    "fmt"

    "NatsumeAI/app/api/coupon/internal/config"
    "NatsumeAI/app/api/coupon/internal/handler"
    "NatsumeAI/app/api/coupon/internal/svc"

    "github.com/zeromicro/go-zero/core/conf"
    "github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/coupon-api.yaml", "the config file")

func main() {
    flag.Parse()

    var c config.Config
    conf.MustLoad(*configFile, &c)

    server := rest.MustNewServer(c.RestConf)
    defer server.Stop()

    ctx := svc.NewServiceContext(c)
    handler.RegisterHandlers(server, ctx)

    fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
    server.Start()
}

