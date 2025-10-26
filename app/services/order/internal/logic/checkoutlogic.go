package logic

import (
	"context"
	"time"

	"NatsumeAI/app/common/consts/errno"
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

    // 生成预订单 id 和过期时间（统一使用全局 TTL）
    expireAt := time.Now().Add(l.svcCtx.PreorderTTL)
    preorderID := snowflake.Next()

    // 第一步：尝试获取令牌（快速失败/成功），需检查返回码
    if ir, err := l.svcCtx.Inventory.TryGetToken(l.ctx, &invpb.TryGetTokenReq{
        PreorderId: preorderID,
        Item: &invpb.Item{ProductId: in.Item.ProductId, Quantity: in.Item.Quantity},
    }); err != nil {
        // RPC 层失败
        resp.StatusCode = 500
        resp.StatusMsg = err.Error()
        return resp, nil
    } else if ir != nil && ir.StatusCode != errno.StatusOK {
        // 业务失败（如 NOT_ENOUGH、INVALID_*）
        resp.StatusCode = 409
        if ir.StatusMsg != "" { resp.StatusMsg = ir.StatusMsg } else { resp.StatusMsg = "insufficient inventory" }
        return resp, nil
    }

    // 第二步：拉取商品价格（若商品服务不可用，则退化为 0）
    var priceCents int64
    var snap *mq.CheckoutSnapshot
    if l.svcCtx.Product != nil {
        if pr, err := l.svcCtx.Product.GetProduct(l.ctx, &prodpb.GetProductReq{
            ProductId: in.Item.ProductId,
            UserId:    in.UserId,
        }); err == nil && pr != nil && pr.Product != nil {
            priceCents = pr.Product.Price
            snap = &mq.CheckoutSnapshot{
                Title:      pr.Product.Name,
                CoverImage: pr.Product.Picture,
                Attributes: pr.Product.Description,
            }
        } else {
            // 获取商品失败：归还令牌并告知客户端
            _, _ = l.svcCtx.Inventory.ReturnToken(l.ctx, &invpb.ReturnTokenReq{
                PreorderId: preorderID,
                Item:       &invpb.Item{ProductId: in.Item.ProductId, Quantity: in.Item.Quantity},
            })
            resp.StatusCode = 404
            resp.StatusMsg = "product not found"
            return resp, nil
        }
    }

    // 计算金额
    totalAmount := priceCents * in.Item.Quantity
    finalAmount := totalAmount

    // 第三步：若传入优惠券，先校验并锁券
    lockedCoupon := int64(0)
    if in.CouponId > 0 && l.svcCtx.Coupon != nil && totalAmount > 0 {
        v, err := l.svcCtx.Coupon.ValidateCoupon(l.ctx, &couponsvcpb.ValidateCouponReq{
            UserId:      in.UserId,
            CouponId:    in.CouponId,
            OrderAmount: totalAmount,
        })
        if err != nil || v == nil || !v.Valid {
            // 归还令牌
            _, _ = l.svcCtx.Inventory.ReturnToken(l.ctx, &invpb.ReturnTokenReq{
                PreorderId: preorderID,
                Item:       &invpb.Item{ProductId: in.Item.ProductId, Quantity: in.Item.Quantity},
            })
            resp.StatusCode = 400
            resp.StatusMsg = "invalid coupon"
            return resp, nil
        }
        if v.DiscountAmount > 0 {
            finalAmount = totalAmount - v.DiscountAmount
            if finalAmount < 0 { finalAmount = 0 }
        }
        if _, err := l.svcCtx.Coupon.LockCoupon(l.ctx, &couponsvcpb.LockCouponReq{
            UserId:   in.UserId,
            CouponId: in.CouponId,
            OrderId:  preorderID,
        }); err != nil {
            // 归还令牌
            _, _ = l.svcCtx.Inventory.ReturnToken(l.ctx, &invpb.ReturnTokenReq{
                PreorderId: preorderID,
                Item:       &invpb.Item{ProductId: in.Item.ProductId, Quantity: in.Item.Quantity},
            })
            resp.StatusCode = 409
            resp.StatusMsg = "coupon lock failed"
            return resp, nil
        }
        lockedCoupon = in.CouponId
    }

    // 第四步：写入预订单（供后续 Place 使用）
    po := &orderdal.OrderPreorders{
        PreorderId:     preorderID,
        UserId:         in.UserId,
        CouponId:       lockedCoupon,
        OriginalAmount: totalAmount,
        FinalAmount:    finalAmount,
        Status:         "PENDING",
        ExpireAt:       expireAt,
    }
    if _, err := l.svcCtx.Preorder.InsertWithId(l.ctx, po); err != nil {
        // 回滚（令牌 + 优惠券）
        _, _ = l.svcCtx.Inventory.ReturnToken(l.ctx, &invpb.ReturnTokenReq{
            PreorderId: preorderID,
            Item:       &invpb.Item{ProductId: in.Item.ProductId, Quantity: in.Item.Quantity},
        })
        if lockedCoupon > 0 && l.svcCtx.Coupon != nil {
            _, _ = l.svcCtx.Coupon.ReleaseCoupon(l.ctx, &couponsvcpb.ReleaseCouponReq{
                UserId:   in.UserId,
                CouponId: lockedCoupon,
                OrderId:  preorderID,
            })
        }
        resp.StatusCode = 500
        resp.StatusMsg = "create preorder failed"
        return resp, nil
    }

    // 第五步：发送 checkout 消息，异步冻结库存
    err := mq.PublishCheckoutEvent(l.svcCtx, mq.CheckoutEvent{
        PreorderId: preorderID,
        UserId:     in.UserId,
        CouponId:   lockedCoupon,
        ProductId:  in.Item.ProductId,
        Quantity:   in.Item.Quantity,
        PriceCents: priceCents,
        Snapshot:   snap,
    })
    if err != nil {
        l.Logger.Errorf("publish checkout event failed: preorder=%d err=%v", preorderID, err)
    } else {
        l.Logger.Infof("checkout event enqueued: preorder=%d", preorderID)
    }

    resp.StatusCode = 0
    resp.StatusMsg = "queued"
    resp.PreorderId = preorderID
    resp.ExpiredAt = expireAt.Unix()
    return resp, nil
}
