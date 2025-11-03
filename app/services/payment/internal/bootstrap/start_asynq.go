package bootstrap

import (
	"github.com/hibiken/asynq"

	"NatsumeAI/app/services/payment/internal/mq"
	"NatsumeAI/app/services/payment/internal/svc"
)

func StartAsynq(sc *svc.ServiceContext) func() {
	addr := sc.Config.AsynqConf.Addr
	if addr == "" {
		addr = sc.Config.RedisConf.Host
	}
	srv := asynq.NewServer(asynq.RedisClientOpt{Addr: addr}, asynq.Config{
		Concurrency: sc.Config.AsynqServerConf.Concurrency,
		Queues:      sc.Config.AsynqServerConf.Queues,
	})
	mux := mq.NewAsynqMux(sc)
	go func() {
		if err := srv.Run(mux); err != nil {
			panic(err)
		}
	}()
	return func() {
		srv.Shutdown()
	}
}
