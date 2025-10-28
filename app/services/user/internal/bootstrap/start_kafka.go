package bootstrap

import (
    "context"

    "NatsumeAI/app/services/user/internal/mq"
    "NatsumeAI/app/services/user/internal/svc"
)

// StartKafka starts merchant review consumer if configured; returns a stop func.
func StartKafka(sc *svc.ServiceContext) func() {
    ctx, cancel := context.WithCancel(context.Background())
    go func() { _ = mq.StartMerchantReviewConsumer(ctx, sc) }()
    return func() { cancel() }
}

