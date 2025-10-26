// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package order

import (
    "context"

    "NatsumeAI/app/api/order/internal/svc"
    "NatsumeAI/app/api/order/internal/types"
    "NatsumeAI/app/common/util"
    "NatsumeAI/app/services/order/orderservice"
    helper "NatsumeAI/app/api/order/internal/logic/helper"

    "github.com/zeromicro/go-zero/core/logx"
)

type GetOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrderLogic {
	return &GetOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOrderLogic) GetOrder(req *types.GetOrderRequest) (resp *types.GetOrderResponse, err error) {
    uid, _ := util.UserIdFromCtx(l.ctx)
    in := &orderservice.GetOrderReq{
        OrderId: req.Order_id,
        UserId:  uid,
    }
    out, err := l.svcCtx.OrderRpc.GetOrder(l.ctx, in)
    if err != nil {
        return nil, err
    }
    info := helper.ToOrderInfo(out.Order)
    return &types.GetOrderResponse{
        Status_code: out.StatusCode,
        Status_msg:  out.StatusMsg,
        Order:       info,
    }, nil
}
