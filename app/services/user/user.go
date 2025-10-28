package main

import (
	"flag"
	"fmt"

	boot "NatsumeAI/app/services/user/internal/bootstrap"
	"NatsumeAI/app/services/user/internal/config"
	"NatsumeAI/app/services/user/internal/server"
	"NatsumeAI/app/services/user/internal/svc"
	"NatsumeAI/app/services/user/user"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"github.com/zeromicro/zero-contrib/zrpc/registry/consul"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/user.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

    s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
        user.RegisterUserServiceServer(grpcServer, server.NewUserServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})

    // Start DTM HTTP server if configured; else start outbox publisher
    if stop := boot.StartDTMHTTP(ctx); stop != nil { defer stop() }
    // start MQ consumer (merchant review)
    if stop := boot.StartKafka(ctx); stop != nil { defer stop() }

    if err := consul.RegisterService(c.ListenOn, c.Consul); err != nil {
        logx.Errorw("register service error", logx.Field("err", err))
        panic(err)
    }
    defer s.Stop()
    if ctx.KafkaWriter != nil { defer ctx.KafkaWriter.Close() }

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
