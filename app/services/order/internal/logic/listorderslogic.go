package logic

import (
	"context"

	"NatsumeAI/app/services/order/internal/svc"
	"NatsumeAI/app/services/order/order"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListOrdersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListOrdersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOrdersLogic {
	return &ListOrdersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 分页查询订单
func (l *ListOrdersLogic) ListOrders(in *order.ListOrdersReq) (*order.ListOrdersResp, error) {
    resp := &order.ListOrdersResp{}
    if in == nil || in.UserId <= 0 || in.Page <= 0 || in.PageSize <= 0 {
        resp.StatusCode = 400
        resp.StatusMsg = "invalid params"
        return resp, nil
    }

    offset := (in.Page - 1) * in.PageSize
    rows, err := l.svcCtx.Orders.ListByUser(l.ctx, in.UserId, offset, in.PageSize)
    if err != nil {
        return nil, err
    }
    total, _ := l.svcCtx.Orders.CountByUser(l.ctx, in.UserId)

    orders := make([]*order.OrderInfo, 0, len(rows))
    for _, r := range rows {
        info := &order.OrderInfo{
            OrderId:        r.OrderId,
            PreorderId:     r.PreorderId,
            UserId:         r.UserId,
            Status:         toProtoStatus(r.Status),
            TotalAmount:    r.TotalAmount,
            PayAmount:      r.PaidAmount,
            CreatedAt:      r.CreatedAt.Unix(),
            PaymentMethod:  r.PaymentMethod,
            AddressSnapshot: func() string { if r.AddressSnapshot.Valid { return r.AddressSnapshot.String }; return "" }(),
        }
        orders = append(orders, info)
    }

    resp.StatusCode = 0
    resp.StatusMsg = "ok"
    resp.Orders = orders
    resp.Total = total
    return resp, nil
}
