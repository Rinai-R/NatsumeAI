package logic

import (
    "context"
    "database/sql"
    "encoding/json"
    "time"

    orderdal "NatsumeAI/app/dal/order"
    "NatsumeAI/app/services/order/internal/mq"
    "NatsumeAI/app/services/order/internal/svc"
    "NatsumeAI/app/services/order/order"

    "github.com/zeromicro/go-zero/core/logx"
    "github.com/hibiken/asynq"
    "github.com/zeromicro/go-zero/core/stores/sqlx"
)

type PlaceOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPlaceOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PlaceOrderLogic {
	return &PlaceOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 提交订单（冻结库存，生成正式订单）
func (l *PlaceOrderLogic) PlaceOrder(in *order.PlaceOrderReq) (*order.PlaceOrderResp, error) {
    resp := &order.PlaceOrderResp{}
    if in == nil || in.PreorderId <= 0 || in.UserId <= 0 {
        resp.StatusCode = 400
        resp.StatusMsg = "invalid params"
        return resp, nil
    }
    // 幂等：按 preorder_id 查是否已下过单
    if ord, err := l.svcCtx.Orders.FindOneByPreorderId(l.ctx, in.PreorderId); err == nil && ord != nil {
        resp.StatusCode = 0
        resp.StatusMsg = "ok"
        resp.OrderId = ord.OrderId
        resp.Status = order.OrderStatus_ORDER_STATUS_PENDING
        return resp, nil
    }

    // 读取预订单（确保未过期）
    po, err := l.svcCtx.Preorder.FindOne(l.ctx, in.PreorderId)
    if err != nil {
        return nil, err
    }
    if po.UserId != in.UserId {
        resp.StatusCode = 403
        resp.StatusMsg = "forbidden"
        return resp, nil
    }
    if time.Now().After(po.ExpireAt) || po.Status != "PENDING" {
        resp.StatusCode = 409
        resp.StatusMsg = "preorder expired or invalid"
        return resp, nil
    }

    // 本地事务：原子更新预订单为 PLACED 并插入订单
    var orderID int64
    err = l.svcCtx.DB.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
        // 1. 原子置位预订单（需未过期且仍为 PENDING）
        ok, err := l.svcCtx.Preorder.PlaceIfPendingWithSession(ctx, session, in.PreorderId)
        if err != nil {
            return err
        }
        if !ok {
            return sql.ErrNoRows
        }
        // 2. 插入订单（优惠券以预订单上的为准）
        ord := &orderdal.Orders{
            PreorderId:    in.PreorderId,
            UserId:        in.UserId,
            CouponId:      po.CouponId,
            Status:        "PENDING_PAYMENT",
            TotalAmount:   po.OriginalAmount,
            PayableAmount: po.FinalAmount,
            PaidAmount:    0,
            PaymentMethod: "",
            PaymentAt:     sql.NullTime{},
            ExpireTime:    po.ExpireAt.Unix(),
            CancelReason:  "",
        }
        res, err := l.svcCtx.Orders.InsertWithSession(ctx, session, ord)
        if err != nil {
            return err
        }
        oid, _ := res.LastInsertId()
        orderID = oid
        return nil
    })
    if err != nil {
        if err == sql.ErrNoRows {
            resp.StatusCode = 409
            resp.StatusMsg = "preorder expired or invalid"
            return resp, nil
        }
        return nil, err
    }

    // 锁券已在 Checkout 完成，这里不再重复锁券；核销在 ConfirmPayment。

    // 复制预订单条目到订单条目（单商品场景）
    if rows, err := l.svcCtx.PreItm.ListByPreorder(l.ctx, in.PreorderId); err == nil && len(rows) > 0 {
        for _, it := range rows {
            _, _ = l.svcCtx.OrdItm.Insert(l.ctx, &orderdal.OrderItems{
                OrderId:    uint64(orderID),
                ProductId:  uint64(it.ProductId),
                Quantity:   uint64(it.Quantity),
                PriceCents: uint64(it.PriceCents),
                Snapshot:   it.Snapshot,
            })
        }
    }

    // 安排订单未支付超时取消任务
    if l.svcCtx.AsynqClient != nil {
        delay := time.Until(po.ExpireAt)
        if delay <= 0 { delay = time.Second }
        payload, _ := json.Marshal(mq.CancelOrderTaskPayload{
            OrderId:    orderID,
            PreorderId: in.PreorderId,
            UserId:     in.UserId,
        })
        task := asynq.NewTask(mq.TaskCancelOrder, payload)
        _, _ = l.svcCtx.AsynqClient.Enqueue(task, asynq.ProcessIn(delay), asynq.Queue("default"))
    }

    resp.StatusCode = 0
    resp.StatusMsg = "ok"
    resp.OrderId = orderID
    resp.Status = order.OrderStatus_ORDER_STATUS_PENDING

    // 失败补偿说明：如果后续任一步失败（例如后续步骤需扩展），需：
    // if needRelease { ReleaseCoupon(preorder_id) }
    // 优惠券释放逻辑由取消流程统一处理

    return resp, nil
}
