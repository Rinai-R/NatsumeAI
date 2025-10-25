package logic

import (
	"context"
	"database/sql"
	"time"

	couponsvcpb "NatsumeAI/app/services/coupon/coupon"
	invpb "NatsumeAI/app/services/inventory/inventory"
	"NatsumeAI/app/services/order/internal/svc"
	"NatsumeAI/app/services/order/order"

	"github.com/zeromicro/go-zero/core/logx"
)

type ConfirmPaymentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewConfirmPaymentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfirmPaymentLogic {
	return &ConfirmPaymentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 支付确认（支付成功）
func (l *ConfirmPaymentLogic) ConfirmPayment(in *order.ConfirmPaymentReq) (*order.ConfirmPaymentResp, error) {
    resp := &order.ConfirmPaymentResp{}
    if in == nil || in.OrderId <= 0 || in.UserId <= 0 {
        resp.StatusCode = 400
        resp.StatusMsg = "invalid params"
        return resp, nil
    }

    // 读取订单
    ord, err := l.svcCtx.Orders.FindOne(l.ctx, in.OrderId)
    if err != nil {
        return nil, err
    }
    if ord.Status != "PENDING_PAYMENT" {
        resp.StatusCode = 0
        resp.StatusMsg = "ok"
        resp.Status = order.OrderStatus_ORDER_STATUS_CONFIRMED
        return resp, nil
    }

    // 确认库存（从冻结转已售）
    _, err = l.svcCtx.Inventory.DecreaseInventory(l.ctx, &invpb.DecreaseInventoryReq{
        OrderId:    ord.OrderId,
    })
    if err != nil {
        return nil, err
    }

    // 核销优惠券（可选）
    if ord.CouponId > 0 && l.svcCtx.Coupon != nil {
        _, err = l.svcCtx.Coupon.RedeemCoupon(l.ctx, &couponsvcpb.RedeemCouponReq{
            UserId:      in.UserId,
            CouponId:    ord.CouponId,
            OrderId:     ord.OrderId,
            OrderAmount: ord.PayableAmount,
        })
        if err != nil {
            return nil, err
        }
    }

    // 更新订单状态与支付时间
    ord.Status = "PAID"
    ord.PaymentMethod = in.PaymentMethod
    ord.PaymentAt = sql.NullTime{Time: time.Now(), Valid: true}
    if err := l.svcCtx.Orders.Update(l.ctx, ord); err != nil {
        return nil, err
    }

    resp.StatusCode = 0
    resp.StatusMsg = "ok"
    resp.Status = order.OrderStatus_ORDER_STATUS_CONFIRMED
    return resp, nil
}
