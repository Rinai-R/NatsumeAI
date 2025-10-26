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

type CancelOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCancelOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelOrderLogic {
	return &CancelOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CancelOrderLogic) CancelOrder(req *types.CancelOrderRequest) (resp *types.CancelOrderResponse, err error) {
    uid, _ := util.UserIdFromCtx(l.ctx)
    in := &orderservice.CancelOrderReq{
        PreorderId: req.Preorder_id,
        OrderId:    req.Order_id,
        Reason:     req.Reason,
        UserId:     uid,
    }
    out, err := l.svcCtx.OrderRpc.CancelOrder(l.ctx, in)
    if err != nil {
        return nil, err
    }
    return &types.CancelOrderResponse{
        Status_code: out.StatusCode,
        Status_msg:  out.StatusMsg,
        Status:      int32(out.Status),
    }, nil
}
