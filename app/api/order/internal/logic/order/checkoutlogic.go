// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package order

import (
    "context"

    "NatsumeAI/app/api/order/internal/svc"
    "NatsumeAI/app/api/order/internal/types"
    "NatsumeAI/app/common/util"
    "NatsumeAI/app/services/order/orderservice"

    "github.com/zeromicro/go-zero/core/logx"
)

type CheckoutLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCheckoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckoutLogic {
	return &CheckoutLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CheckoutLogic) Checkout(req *types.CheckoutRequest) (resp *types.CheckoutResponse, err error) {
    // 强制使用中间件注入的 userId
    uid, _ := util.UserIdFromCtx(l.ctx)
    in := &orderservice.CheckoutReq{
        UserId:   uid,
        CouponId: req.Coupon_id,
        Item: &orderservice.Item{
            ProductId: req.Item.Product_id,
            Quantity:  req.Item.Quantity,
        },
    }
    out, err := l.svcCtx.OrderRpc.Checkout(l.ctx, in)
    if err != nil {
        return nil, err
    }
    return &types.CheckoutResponse{
        Status_code: out.StatusCode,
        Status_msg:  out.StatusMsg,
        Preorder_id: out.PreorderId,
        Expired_at:  out.ExpiredAt,
    }, nil
}
