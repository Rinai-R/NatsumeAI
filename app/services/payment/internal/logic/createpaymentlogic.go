package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"NatsumeAI/app/common/snowflake"
	paymentdal "NatsumeAI/app/dal/payment"
	"NatsumeAI/app/services/order/order"
	"NatsumeAI/app/services/payment/internal/mq"
	"NatsumeAI/app/services/payment/internal/svc"
	paymentpb "NatsumeAI/app/services/payment/payment"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreatePaymentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreatePaymentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreatePaymentLogic {
	return &CreatePaymentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreatePaymentLogic) CreatePayment(in *paymentpb.CreatePaymentReq) (*paymentpb.CreatePaymentResp, error) {
	resp := &paymentpb.CreatePaymentResp{}
	if in == nil || in.OrderId <= 0 || in.UserId <= 0 || in.Amount <= 0 || in.Channel == "" {
		resp.StatusCode = 400
		resp.StatusMsg = "invalid params"
		return resp, nil
	}
	if in.Currency == "" {
		in.Currency = "CNY"
	}

	// idempotence: return existing payment order if present
	if existing, err := l.svcCtx.PaymentOrders.FindOneByOrderId(l.ctx, in.OrderId); err == nil {
		if existing.UserId != in.UserId {
			resp.StatusCode = 403
			resp.StatusMsg = "forbidden"
			return resp, nil
		}
		resp.StatusCode = 0
		resp.StatusMsg = "ok"
		resp.Payment = toPaymentInfo(existing)
		return resp, nil
	} else if err != paymentdal.ErrNotFound {
		return nil, err
	}

	// 做标记
	markResp, err := l.svcCtx.OrderRpc.MarkPaying(l.ctx, &order.MarkPayingReq{
		OrderId: in.OrderId,
		UserId:  in.UserId,
	})
	if err != nil {
		return nil, err
	}
	if markResp.GetStatusCode() != 0 {
		resp.StatusCode = markResp.StatusCode
		resp.StatusMsg = markResp.StatusMsg
		return resp, nil
	}

	channel := strings.ToUpper(in.Channel)
	timeoutAt := time.Now().Add(l.svcCtx.PaymentTTL)

	payloadData := map[string]string{
		"client_ip": in.ClientIp,
		"subject":   in.Subject,
	}
	payloadBytes, _ := json.Marshal(payloadData)

	record := &paymentdal.PaymentOrders{
		PaymentNo: strconv.FormatInt(snowflake.Next(), 10),
		OrderId:   in.OrderId,
		UserId:    in.UserId,
		Amount:    in.Amount,
		Currency:  strings.ToUpper(in.Currency),
		Channel:   channel,
		Status:    "INIT",
		ChannelPayload: sql.NullString{
			String: string(payloadBytes),
			Valid:  len(payloadBytes) > 2,
		},
		TimeoutAt: timeoutAt,
		Extra:     sql.NullString{},
	}

	res, err := l.svcCtx.PaymentOrders.Insert(l.ctx, record)
	if err != nil {
		if existing, findErr := l.svcCtx.PaymentOrders.FindOneByOrderId(l.ctx, in.OrderId); findErr == nil {
			resp.StatusCode = 0
			resp.StatusMsg = "ok"
			resp.Payment = toPaymentInfo(existing)
			return resp, nil
		} else if findErr != paymentdal.ErrNotFound {
			return nil, findErr
		}
		return nil, err
	}
	paymentID, _ := res.LastInsertId()
	record.PaymentId = paymentID

	if l.svcCtx.AsynqClient != nil {
		payload, _ := json.Marshal(mq.ExpirePaymentPayload{
			PaymentId: paymentID,
			OrderId:   in.OrderId,
			UserId:    in.UserId,
		})
		task := asynq.NewTask(mq.TaskExpirePayment, payload)
		if _, err := l.svcCtx.AsynqClient.Enqueue(task, asynq.ProcessIn(l.svcCtx.PaymentTTL), asynq.Queue("payment")); err != nil {
			l.Logger.Errorf("enqueue payment timeout failed: payment=%d err=%v", paymentID, err)
		}
	}

	resp.StatusCode = 0
	resp.StatusMsg = "ok"
	record.Status = "INIT"
	resp.Payment = toPaymentInfo(record)
	return resp, nil
}
