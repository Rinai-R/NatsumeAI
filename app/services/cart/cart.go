package main

import (
	"flag"
	"fmt"

	"NatsumeAI/app/services/cart/cart"
	"NatsumeAI/app/services/cart/internal/config"
	"NatsumeAI/app/services/cart/internal/server"
	"NatsumeAI/app/services/cart/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"github.com/zeromicro/zero-contrib/zrpc/registry/consul"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/cart.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		cart.RegisterCartServiceServer(grpcServer, server.NewCartServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})

	if err := consul.RegisterService(c.ListenOn, c.Consul); err != nil {
		logx.Errorw("register service error", logx.Field("err", err))
		panic(err)
	}
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
