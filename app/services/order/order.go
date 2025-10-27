package main

import (
	"flag"
	"fmt"

	boot "NatsumeAI/app/services/order/internal/bootstrap"
	"NatsumeAI/app/services/order/internal/config"
	"NatsumeAI/app/services/order/internal/server"
	"NatsumeAI/app/services/order/internal/svc"
	"NatsumeAI/app/services/order/order"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"github.com/zeromicro/zero-contrib/zrpc/registry/consul"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/order.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
    ctx := svc.NewServiceContext(c)

    s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
        order.RegisterOrderServiceServer(grpcServer, server.NewOrderServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
    })

    // Start Kafka consumer if configured (no-op unless built with kafka tag)
    if stop := boot.StartKafka(ctx); stop != nil {
        defer stop() 
    }
    // Start DTM HTTP callbacks server if configured
    if stop := boot.StartDTMHTTP(ctx); stop != nil {
        defer stop()
    }

	if err := consul.RegisterService(c.ListenOn, c.Consul); err != nil {
		logx.Errorw("register service error", logx.Field("err", err))
		panic(err)
	}
    defer s.Stop()
    if ctx.KafkaWriter != nil {
        defer ctx.KafkaWriter.Close()
    }

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
