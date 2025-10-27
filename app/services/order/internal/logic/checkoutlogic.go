package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
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

	"github.com/dtm-labs/dtmcli"
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

// Checkout 发布下单事件，由异步消费者扣库存，锁定优惠券等操作
func (l *CheckoutLogic) Checkout(in *order.CheckoutReq) (*order.CheckoutResp, error) {
    resp := &order.CheckoutResp{}
    if in == nil || in.UserId <= 0 || in.Item == nil || in.Item.ProductId <= 0 || in.Item.Quantity <= 0 {
        resp.StatusCode = 400
        resp.StatusMsg = "invalid params"
        return resp, nil
    }


    expireAt := time.Now().Add(l.svcCtx.PreorderTTL)
    preorderID := snowflake.Next()

    // 根据库存量获取 token（redis 快速过滤）
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

    couponId := int64(0)
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
        couponId = in.CouponId
    }

    // 第四步：写入预订单 + 可靠发布消息（优先使用 DTM）
    evt := mq.CheckoutEvent{
        PreorderId: preorderID,
        UserId:     in.UserId,
        CouponId:   couponId,
        ProductId:  in.Item.ProductId,
        Quantity:   in.Item.Quantity,
        PriceCents: priceCents,
        Snapshot:   snap,
    }
    if l.svcCtx.Config.DtmConf.Server != "" && l.svcCtx.Config.DtmConf.BusiURL != "" {
        gid := "checkout-" + strconv.FormatInt(preorderID, 10)
        body, _ := json.Marshal(evt)
        msg := dtmcli.NewMsg(l.svcCtx.Config.DtmConf.Server, gid).
            Add(l.svcCtx.Config.DtmConf.BusiURL+"/dtm/checkout/publish", body)
        qp := l.svcCtx.Config.DtmConf.BusiURL + "/dtm/checkout/query?preorder_id=" + strconv.FormatInt(preorderID, 10)
        if err := msg.DoAndSubmitDB(qp, l.svcCtx.RawDB, func(tx *sql.Tx) error {
            // 使用原生 SQL 写入预订单，确保与消息提交原子
            query := "insert into `order_preorders` (`preorder_id`,`user_id`,`coupon_id`,`original_amount`,`final_amount`,`status`,`expire_at`) values (?,?,?,?,?,?,?)"
            _, err := tx.ExecContext(l.ctx, query, preorderID, in.UserId, couponId, totalAmount, finalAmount, "PENDING", expireAt)
            return err
        }); err != nil {
            // 回滚令牌
            _, _ = l.svcCtx.Inventory.ReturnToken(l.ctx, &invpb.ReturnTokenReq{
                PreorderId: preorderID,
                Item:       &invpb.Item{ProductId: in.Item.ProductId, Quantity: in.Item.Quantity},
            })
            resp.StatusCode = 500
            resp.StatusMsg = "create preorder failed"
            return resp, nil
        }
        l.Logger.Infof("checkout msg submitted via dtm: preorder=%d", preorderID)
    } else {
        // 回退方案：直接插入 + 直接发 Kafka
        po := &orderdal.OrderPreorders{
            PreorderId:     preorderID,
            UserId:         in.UserId,
            CouponId:       couponId,
            OriginalAmount: totalAmount,
            FinalAmount:    finalAmount,
            Status:         "PENDING",
            ExpireAt:       expireAt,
        }
        if _, err := l.svcCtx.Preorder.InsertWithId(l.ctx, po); err != nil {
            // 回滚令牌
            _, _ = l.svcCtx.Inventory.ReturnToken(l.ctx, &invpb.ReturnTokenReq{
                PreorderId: preorderID,
                Item:       &invpb.Item{ProductId: in.Item.ProductId, Quantity: in.Item.Quantity},
            })
            resp.StatusCode = 500
            resp.StatusMsg = "create preorder failed"
            return resp, nil
        }
        if err := mq.PublishCheckoutEvent(l.svcCtx, evt); err != nil {
            l.Logger.Errorf("publish checkout event failed: preorder=%d err=%v", preorderID, err)
        } else {
            l.Logger.Infof("checkout event enqueued: preorder=%d", preorderID)
        }
    }

    resp.StatusCode = 0
    resp.StatusMsg = "queued"
    resp.PreorderId = preorderID
    resp.ExpiredAt = expireAt.Unix()
    return resp, nil
}
