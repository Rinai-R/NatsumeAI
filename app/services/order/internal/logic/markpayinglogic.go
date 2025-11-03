package logic

import (
	"context"

	"NatsumeAI/app/services/order/internal/svc"
	"NatsumeAI/app/services/order/order"

	"github.com/zeromicro/go-zero/core/logx"
)

type MarkPayingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMarkPayingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkPayingLogic {
	return &MarkPayingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// MarkPaying sets the order status to PAYING so only the payment service manages timeout.
func (l *MarkPayingLogic) MarkPaying(in *order.MarkPayingReq) (*order.MarkPayingResp, error) {
	resp := &order.MarkPayingResp{}
	if in == nil || in.OrderId <= 0 || in.UserId <= 0 {
		resp.StatusCode = 400
		resp.StatusMsg = "invalid params"
		return resp, nil
	}

	ord, err := l.svcCtx.Orders.FindOne(l.ctx, in.OrderId)
	if err != nil {
		return nil, err
	}
	if ord.UserId != in.UserId {
		resp.StatusCode = 403
		resp.StatusMsg = "forbidden"
		return resp, nil
	}

	switch ord.Status {
	case "PAYING":
		resp.StatusCode = 0
		resp.StatusMsg = "ok"
		resp.Status = order.OrderStatus_ORDER_STATUS_PAYING
		return resp, nil
	case "PENDING_PAYMENT":
		// continue
	case "PAID", "COMPLETED":
		resp.StatusCode = 409
		resp.StatusMsg = "order already paid"
		resp.Status = toProtoStatus(ord.Status)
		return resp, nil
	case "CANCELLED":
		resp.StatusCode = 409
		resp.StatusMsg = "order already cancelled"
		resp.Status = toProtoStatus(ord.Status)
		return resp, nil
	default:
		resp.StatusCode = 409
		resp.StatusMsg = "invalid order state"
		resp.Status = order.OrderStatus_ORDER_STATUS_UNKNOWN
		return resp, nil
	}

	ord.Status = "PAYING"
	if err := l.svcCtx.Orders.Update(l.ctx, ord); err != nil {
		return nil, err
	}

	resp.StatusCode = 0
	resp.StatusMsg = "ok"
	resp.Status = order.OrderStatus_ORDER_STATUS_PAYING
	return resp, nil
}
