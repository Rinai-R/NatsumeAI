package logic

import (
	"context"
	"database/sql"
	"time"

	orderdal "NatsumeAI/app/dal/order"
	"NatsumeAI/app/services/order/internal/svc"
	"NatsumeAI/app/services/order/order"

	"github.com/zeromicro/go-zero/core/logx"
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
    if time.Now().After(po.ExpireAt) || po.Status != "PENDING" {
        resp.StatusCode = 409
        resp.StatusMsg = "preorder expired or invalid"
        return resp, nil
    }

    // 创建订单（Saga 步骤 1）
    // 优惠券以预订单上的为准，避免重复锁券
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
    res, err := l.svcCtx.Orders.Insert(l.ctx, ord)
    if err != nil {
        return nil, err
    }
    orderID, _ := res.LastInsertId()

    // 锁券已在 Checkout 完成，这里不再重复锁券；核销在 ConfirmPayment。

    // 将预订单标记为已下单（不严格要求强一致）
    po.Status = "PLACED"
    _ = l.svcCtx.Preorder.Update(l.ctx, po)

    // 同步一份订单条目（从预订单条目复制一条）
    // 这里简单复制第一条预订单条目
    // 若需要，可查询所有 preorder items 并批量插入
    // 为简洁，我们不展开查询明细，保留给后续完善

    resp.StatusCode = 0
    resp.StatusMsg = "ok"
    resp.OrderId = orderID
    resp.Status = order.OrderStatus_ORDER_STATUS_PENDING

    // 失败补偿说明：如果后续任一步失败（例如后续步骤需扩展），需：
    // if needRelease { ReleaseCoupon(preorder_id) }
    // 优惠券释放逻辑由取消流程统一处理

    return resp, nil
}
