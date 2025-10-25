package bootstrap

import (
	"NatsumeAI/app/services/order/internal/mq"
	"NatsumeAI/app/services/order/internal/svc"
	"context"
	"time"

	"github.com/hibiken/asynq"
)

// StartKafka starts Kafka consumer and Asynq server; returns a stop func.
func StartKafka(sc *svc.ServiceContext) func() {
    // asynq 延时队列
    redisOpt := sc.Config.AsynqConf
    if redisOpt.Addr == "" {
        redisOpt = asynq.RedisClientOpt{Addr: sc.Config.RedisConf.Host, Password: sc.Config.RedisConf.Pass}
    }
    srv := asynq.NewServer(redisOpt, sc.Config.AsynqServerConf)
    mux := mq.NewAsynqMux(sc)
    go func() { _ = srv.Run(mux) }()

    // kafka 消费者
    ctx, cancel := context.WithCancel(context.Background())
    go func() { _ = mq.StartCheckoutConsumer(ctx, sc) }()

    return func() {
        cancel()
        srv.Shutdown()
        time.Sleep(100 * time.Millisecond)
    }
}
