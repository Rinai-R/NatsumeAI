package logic

import (
    "context"
    "encoding/json"

    "NatsumeAI/app/services/order/internal/svc"
    "NatsumeAI/app/services/order/order"

    "github.com/zeromicro/go-zero/core/logx"
)

type GetOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrderLogic {
	return &GetOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询单个订单
func (l *GetOrderLogic) GetOrder(in *order.GetOrderReq) (*order.GetOrderResp, error) {
    resp := &order.GetOrderResp{}
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

    info := &order.OrderInfo{
        OrderId:        ord.OrderId,
        PreorderId:     ord.PreorderId,
        UserId:         ord.UserId,
        Status:         toProtoStatus(ord.Status),
        TotalAmount:    ord.TotalAmount,
        PayAmount:      ord.PaidAmount,
        CreatedAt:      ord.CreatedAt.Unix(),
        PaidAt:         0,
        CancelledAt:    0,
        PaymentMethod:  ord.PaymentMethod,
        AddressSnapshot: func() string { if ord.AddressSnapshot.Valid { return ord.AddressSnapshot.String }; return "" }(),
    }
    if ord.PaymentAt.Valid { info.PaidAt = ord.PaymentAt.Time.Unix() }
    // load items from order_items
    items, _ := l.listOrderItems(ord.OrderId)
    info.Items = items

    resp.StatusCode = 0
    resp.StatusMsg = "ok"
    resp.Order = info
    return resp, nil
}

func (l *GetOrderLogic) listOrderItems(orderID int64) ([]*order.OrderItem, error) {
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
