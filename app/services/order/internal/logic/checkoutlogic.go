package logic

import (
	"context"
	"encoding/json"
	"time"

	"NatsumeAI/app/common/snowflake"
	orderdal "NatsumeAI/app/dal/order"
	couponsvcpb "NatsumeAI/app/services/coupon/coupon"
	invpb "NatsumeAI/app/services/inventory/inventory"
	"NatsumeAI/app/services/order/internal/mq"
	"NatsumeAI/app/services/order/internal/svc"
	"NatsumeAI/app/services/order/order"
	prodpb "NatsumeAI/app/services/product/product"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckoutLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCheckoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckoutLogic {
	return &CheckoutLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Checkout 发布下单事件，由异步消费者创建预订单并预扣库存
func (l *CheckoutLogic) Checkout(in *order.CheckoutReq) (*order.CheckoutResp, error) {
    resp := &order.CheckoutResp{}
    if in == nil || in.UserId <= 0 || in.Item == nil || in.Item.ProductId <= 0 || in.Item.Quantity <= 0 {
        resp.StatusCode = 400
        resp.StatusMsg = "invalid params"
        return resp, nil
    }

    // 雪花预订单 id
    expireAt := time.Now().Add(30 * time.Minute)
    preorderID := snowflake.Next()
    // 获取 token
    if _, err := l.svcCtx.Inventory.TryGetToken(l.ctx, &invpb.TryGetTokenReq{PreorderId: preorderID, Item: &invpb.Item{ProductId: in.Item.ProductId, Quantity: in.Item.Quantity}}); err != nil {
        resp.StatusCode = 409
        resp.StatusMsg = "insufficient inventory"
        return resp, nil
    }

    // 获取信息
    var priceCents int64
    var snap *mq.CheckoutSnapshot
    if l.svcCtx.Product != nil {
        if pr, err := l.svcCtx.Product.GetProduct(l.ctx, &prodpb.GetProductReq{ProductId: in.Item.ProductId, UserId: in.UserId}); err == nil && pr != nil && pr.Product != nil {
            priceCents = pr.Product.Price
            snap = &mq.CheckoutSnapshot{Title: pr.Product.Name, CoverImage: pr.Product.Picture, Attributes: pr.Product.Description}
        }
    }

    // 计算优惠券
    originalAmount := priceCents * in.Item.Quantity
    finalAmount := originalAmount
    couponId := in.CouponId
    if couponId > 0 && l.svcCtx.Coupon != nil && originalAmount > 0 {
        if v, err := l.svcCtx.Coupon.ValidateCoupon(l.ctx, &couponsvcpb.ValidateCouponReq{UserId: in.UserId, CouponId: couponId, OrderAmount: originalAmount}); err == nil && v != nil && v.Valid {
            if v.DiscountAmount > 0 && v.DiscountAmount < originalAmount {
                finalAmount = originalAmount - v.DiscountAmount
            }
            // 锁定优惠券
            if _, err := l.svcCtx.Coupon.LockCoupon(l.ctx, &couponsvcpb.LockCouponReq{UserId: in.UserId, CouponId: couponId, OrderId: preorderID}); err != nil {
                couponId = 0
                finalAmount = originalAmount
            }
        } else {
            couponId = 0
        }
    }

    // 创建预订单
    po := &orderdal.OrderPreorders{PreorderId: preorderID, UserId: in.UserId, CouponId: couponId, OriginalAmount: originalAmount, FinalAmount: finalAmount, Status: "PENDING", ExpireAt: expireAt}
    if _, err := l.svcCtx.Preorder.InsertWithId(l.ctx, po); err != nil {
        // 归还 token
        _, _ = l.svcCtx.Inventory.ReturnToken(l.ctx, &invpb.ReturnTokenReq{PreorderId: preorderID, Item: []*invpb.Item{{ProductId: in.Item.ProductId, Quantity: in.Item.Quantity}}})
        return nil, err
    }

    // 消息队列，异步冻结库存
    _ = mq.PublishCheckoutEvent(l.svcCtx, mq.CheckoutEvent{
        PreorderId: preorderID,
        UserId:     in.UserId,
        ProductId:  in.Item.ProductId,
        Quantity:   in.Item.Quantity,
        PriceCents: priceCents,
        Snapshot:   snap,
    })

    resp.StatusCode = 0
    resp.StatusMsg = "queued"
    resp.PreorderId = preorderID
    resp.ExpiredAt = expireAt.Unix()
    resp.OriginalAmount = 0
    resp.FinalAmount = 0
    var snapJSON string
    if snap != nil { if b, _ := json.Marshal(snap); len(b) > 0 { snapJSON = string(b) } }
    resp.LockedItem = &order.OrderItem{ProductId: in.Item.ProductId, Quantity: in.Item.Quantity, PriceCents: priceCents, Snapshot: &order.OrderItemSnapshot{Title: snap.Title, CoverImage: snap.CoverImage, Attributes: snap.Attributes}}
    _ = snapJSON
    return resp, nil
}
