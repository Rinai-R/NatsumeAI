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

type PlaceOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPlaceOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PlaceOrderLogic {
	return &PlaceOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PlaceOrderLogic) PlaceOrder(req *types.PlaceOrderRequest) (resp *types.PlaceOrderResponse, err error) {
    uid, _ := util.UserIdFromCtx(l.ctx)
    in := &orderservice.PlaceOrderReq{
        PreorderId: req.Preorder_id,
        UserId:     uid,
        AddressId:  req.Address_id,
        CouponId:   req.Coupon_id,
        Remark:     req.Remark,
    }
    out, err := l.svcCtx.OrderRpc.PlaceOrder(l.ctx, in)
    if err != nil {
        return nil, err
    }
    return &types.PlaceOrderResponse{
        Status_code: out.StatusCode,
        Status_msg:  out.StatusMsg,
        Order_id:    out.OrderId,
        Status:      int32(out.Status),
    }, nil
}
