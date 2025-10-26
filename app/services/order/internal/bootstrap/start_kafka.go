package bootstrap

import (
	"context"
	"time"

	"NatsumeAI/app/services/order/internal/mq"
	"NatsumeAI/app/services/order/internal/svc"

	"github.com/hibiken/asynq"
)

// StartKafka starts Kafka consumer and Asynq server; returns a stop func.
func StartKafka(sc *svc.ServiceContext) func() {
    // asynq 延时队列（无需密码）
    addr := sc.Config.AsynqConf.Addr
    if addr == "" {
        addr = sc.Config.RedisConf.Host
    }
    redisOpt := asynq.RedisClientOpt{Addr: addr}
    srv := asynq.NewServer(redisOpt, asynq.Config{
        Concurrency: sc.Config.AsynqServerConf.Concurrency,
        Queues:      sc.Config.AsynqServerConf.Queues,
    })
    mux := mq.NewAsynqMux(sc)
    go func() {
        if err := srv.Run(mux); err != nil {
            panic(err)
        }
    }()

    // kafka 消费者
    ctx, cancel := context.WithCancel(context.Background())
    go func() { 
        if err := mq.StartCheckoutConsumer(ctx, sc); err != nil {
            panic(err)
        } 

    }()

    return func() {
        cancel()
        srv.Shutdown()
        time.Sleep(100 * time.Millisecond)
    }
}
