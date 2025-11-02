package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"NatsumeAI/app/services/indexer/internal/config"
	"NatsumeAI/app/services/indexer/internal/mq"
	"NatsumeAI/app/services/indexer/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/sync/errgroup"
)

var configFile = flag.String("f", "etc/indexer.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	group, groupCtx := errgroup.WithContext(rootCtx)
	group.Go(func() error { return mq.StartCanalProductConsumer(groupCtx, ctx) })
	group.Go(func() error { return mq.StartCanalProductCategoryConsumer(groupCtx, ctx) })

	if err := group.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		logx.Errorw("indexer stopped with error", logx.Field("err", err))
		os.Exit(1)
	}

	logx.Info("indexer shutdown gracefully")
}
