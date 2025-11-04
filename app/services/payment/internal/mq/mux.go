package mq

import (
	"NatsumeAI/app/services/payment/internal/svc"

	"github.com/hibiken/asynq"
)

func NewAsynqMux(sc *svc.ServiceContext) *asynq.ServeMux {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskExpirePayment, newExpirePaymentHandler(sc))
	return mux
}
