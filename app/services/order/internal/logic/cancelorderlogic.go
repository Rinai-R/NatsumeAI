package logic

import (
	"context"
	"database/sql"

	"NatsumeAI/app/common/consts/errno"
	couponsvcpb "NatsumeAI/app/services/coupon/coupon"
	invpb "NatsumeAI/app/services/inventory/inventory"
	"NatsumeAI/app/services/order/internal/svc"
	"NatsumeAI/app/services/order/order"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCancelOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelOrderLogic {
	return &CancelOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 主动取消或超时取消订单
func (l *CancelOrderLogic) CancelOrder(in *order.CancelOrderReq) (*order.CancelOrderResp, error) {
    resp := &order.CancelOrderResp{}
    if in == nil || (in.GetOrderId() <= 0 && in.GetPreorderId() <= 0) {
        resp.StatusCode = 400
        resp.StatusMsg = "invalid params"
        return resp, nil
    }

    orderId := in.GetOrderId()
    preorderId := in.GetPreorderId()

    if orderId > 0 {
        ord, err := l.svcCtx.Orders.FindOne(l.ctx, orderId)
        if err != nil {
            return nil, err
        }
        if in.GetUserId() > 0 && ord.UserId != in.GetUserId() {
            resp.StatusCode = 403
            resp.StatusMsg = "forbidden"
            return resp, nil
        }
        preorderId = ord.PreorderId
        // 仅允许未支付订单取消
        if ord.Status == "PAID" || ord.Status == "COMPLETED" {
            resp.StatusCode = 409
            resp.StatusMsg = "order already paid or completed"
            return resp, nil
        }

        // 回滚预占库存、释放券、归还令牌
        l.rollbackPreorderResources(preorderId, in.GetUserId(), ord.CouponId)

        // 更新订单状态
        ord.Status = "CANCELLED"
        ord.CancelReason = in.GetReason()
        ord.PaymentAt = sql.NullTime{}
        if err := l.svcCtx.Orders.Update(l.ctx, ord); err != nil {
            return nil, err
        }

        resp.StatusCode = 0
        resp.StatusMsg = "ok"
        resp.Status = order.OrderStatus_ORDER_STATUS_CANCELLED
        return resp, nil
    }

    // 预订单取消（未生成订单）
    if preorderId <= 0 {
        resp.StatusCode = 400
        resp.StatusMsg = "missing preorder_id"
        return resp, nil
    }

    po, err := l.svcCtx.Preorder.FindOne(l.ctx, preorderId)
    if err != nil {
        return nil, err
    }
    if in.GetUserId() > 0 && po.UserId != in.GetUserId() {
        resp.StatusCode = 403
        resp.StatusMsg = "forbidden"
        return resp, nil
    }

    l.rollbackPreorderResources(preorderId, in.GetUserId(), po.CouponId)

    // 标记预订单取消
    po.Status = "CANCELLED"
    if err := l.svcCtx.Preorder.Update(l.ctx, po); err != nil {
        return nil, err
    }

    resp.StatusCode = 0
    resp.StatusMsg = "ok"
    resp.Status = order.OrderStatus_ORDER_STATUS_CANCELLED
    return resp, nil
}

// rollbackPreorderResources releases frozen inventory, coupon (if any), and returns token.
func (l *CancelOrderLogic) rollbackPreorderResources(preorderId int64, userId int64, couponId int64) {
    // 获取单商品条目（系统为单商品），用于库存 RPC 的单 item 参数
    var item *invpb.Item
    if rows, err := l.svcCtx.PreItm.ListByPreorder(l.ctx, preorderId); err == nil && len(rows) > 0 {
        item = &invpb.Item{
            ProductId: rows[0].ProductId,
            Quantity:  rows[0].Quantity,
        }
    }
    if item == nil {
        item = &invpb.Item{
            ProductId: 0,
            Quantity:  0,
        }
    }

    // 回滚预扣库存
    if rp, err := l.svcCtx.Inventory.ReturnPreInventory(l.ctx, &invpb.InventoryReq{
        OrderId:    preorderId,
        PreorderId: preorderId,
        Item:       item,
    }); err != nil {
        l.Logger.Errorf("cancel rollback pre-inventory failed: preorder=%d product=%d qty=%d err=%v", preorderId, item.ProductId, item.Quantity, err)
    } else if rp != nil && rp.StatusCode != errno.StatusOK {
        l.Logger.Infof("cancel rollback pre-inventory status: preorder=%d code=%d msg=%s", preorderId, rp.StatusCode, rp.StatusMsg)
    }

    // 释放优惠券
    if couponId > 0 && l.svcCtx.Coupon != nil {
        _, _ = l.svcCtx.Coupon.ReleaseCoupon(l.ctx, &couponsvcpb.ReleaseCouponReq{
            UserId:  userId,
            CouponId: couponId,
            OrderId: preorderId,
        })
    }
}
