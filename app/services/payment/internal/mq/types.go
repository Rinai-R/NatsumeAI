package mq

import (
	"context"
	"encoding/json"
	"time"

	"NatsumeAI/app/services/order/order"
	"NatsumeAI/app/services/payment/internal/svc"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
)

const TaskExpirePayment = "payment:expire_payment"

type ExpirePaymentPayload struct {
	PaymentId int64 `json:"payment_id"`
	OrderId   int64 `json:"order_id"`
	UserId    int64 `json:"user_id"`
}

func NewAsynqMux(sc *svc.ServiceContext) *asynq.ServeMux {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskExpirePayment, expirePaymentHandler(sc))
	return mux
}

func expirePaymentHandler(sc *svc.ServiceContext) asynq.HandlerFunc {
	return func(ctx context.Context, task *asynq.Task) error {
		var payload ExpirePaymentPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return err
		}

		po, err := sc.PaymentOrders.FindOne(ctx, payload.PaymentId)
		if err != nil {
			return nil
		}
		switch po.Status {
		case "SUCCESS", "FAILED", "CANCELLED", "EXPIRED":
			return nil
		}
		if time.Now().Before(po.TimeoutAt) {
			return nil
		}

		updated, err := sc.PaymentOrders.UpdateStatus(ctx, po.PaymentId, []string{"INIT", "PROCESSING"}, "EXPIRED")
		if err != nil {
			return err
		}
		if !updated {
			return nil
		}

		_, err = sc.OrderRpc.CancelOrder(ctx, &order.CancelOrderReq{
			OrderId: payload.OrderId,
			UserId:  payload.UserId,
			Reason:  "payment timeout",
		})
		if err != nil {
			logx.WithContext(ctx).Errorf("expire payment cancel order failed: payment=%d order=%d err=%v", payload.PaymentId, payload.OrderId, err)
		}
		return nil
	}
}
