package main

import (
	"flag"
	"fmt"

	"NatsumeAI/app/services/payment/internal/bootstrap"
	"NatsumeAI/app/services/payment/internal/config"
	"NatsumeAI/app/services/payment/internal/server"
	"NatsumeAI/app/services/payment/internal/svc"
	paymentpb "NatsumeAI/app/services/payment/payment"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"github.com/zeromicro/zero-contrib/zrpc/registry/consul"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/payment.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	ctx := svc.NewServiceContext(c)

	srv := zrpc.MustNewServer(c.RpcServerConf, func(gs *grpc.Server) {
		paymentpb.RegisterPaymentServiceServer(gs, server.NewPaymentServiceServer(ctx))
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(gs)
		}
	})

	stopAsynq := bootstrap.StartAsynq(ctx)
	defer stopAsynq()

	if err := consul.RegisterService(c.ListenOn, c.Consul); err != nil {
		logx.Errorw("register service error", logx.Field("err", err))
		panic(err)
	}
	defer srv.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	srv.Start()
}
