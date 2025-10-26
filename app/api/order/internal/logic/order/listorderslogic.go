// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package order

import (
    "context"

    "NatsumeAI/app/api/order/internal/svc"
    "NatsumeAI/app/api/order/internal/types"
    "NatsumeAI/app/common/util"
    "NatsumeAI/app/services/order/orderservice"
    orderpb "NatsumeAI/app/services/order/order"
    helper "NatsumeAI/app/api/order/internal/logic/helper"

    "github.com/zeromicro/go-zero/core/logx"
)

type ListOrdersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListOrdersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOrdersLogic {
	return &ListOrdersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListOrdersLogic) ListOrders(req *types.ListOrdersRequest) (resp *types.ListOrdersResponse, err error) {
    uid, _ := util.UserIdFromCtx(l.ctx)
    in := &orderservice.ListOrdersReq{
        UserId: uid,
        Status: orderpb.OrderStatus(req.Status),
        Page:   req.Page,
        PageSize: req.Page_size,
    }
    out, err := l.svcCtx.OrderRpc.ListOrders(l.ctx, in)
    if err != nil {
        return nil, err
    }
    orders := helper.ToOrderInfos(out.Orders)
    return &types.ListOrdersResponse{
        Status_code: out.StatusCode,
        Status_msg:  out.StatusMsg,
        Orders:      orders,
        Total:       out.Total,
    }, nil
}
