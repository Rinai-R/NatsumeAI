package logic

import (
	"context"
	"encoding/json"

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
        if its, err := l.listOrderItems(r.OrderId); err == nil {
            info.Items = its
        }
        orders = append(orders, info)
    }

    resp.StatusCode = 0
    resp.StatusMsg = "ok"
    resp.Orders = orders
    resp.Total = total
    return resp, nil
}

// listOrderItems loads items from order_items and parses snapshot json.
func (l *ListOrdersLogic) listOrderItems(orderID int64) ([]*order.OrderItem, error) {
    rows, err := l.svcCtx.OrdItm.ListByOrder(l.ctx, orderID)
    if err != nil { return nil, err }
    res := make([]*order.OrderItem, 0, len(rows))
    for _, r := range rows {
        itm := &order.OrderItem{
            ProductId:  int64(r.ProductId),
            Quantity:   int64(r.Quantity),
            PriceCents: int64(r.PriceCents),
        }
        if r.Snapshot.Valid {
            var snap struct{
                Title string `json:"title"`
                CoverImage string `json:"cover_image"`
                Attributes string `json:"attributes"`
            }
            if err := json.Unmarshal([]byte(r.Snapshot.String), &snap); err == nil {
                itm.Snapshot = &order.OrderItemSnapshot{Title: snap.Title, CoverImage: snap.CoverImage, Attributes: snap.Attributes}
            }
        }
        res = append(res, itm)
    }
    return res, nil
}

