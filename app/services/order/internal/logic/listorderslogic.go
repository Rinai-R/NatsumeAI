package logic

import (
	"context"

	"NatsumeAI/app/services/order/internal/svc"
	"NatsumeAI/app/services/order/order"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListOrdersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListOrdersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOrdersLogic {
	return &ListOrdersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 分页查询订单
func (l *ListOrdersLogic) ListOrders(in *order.ListOrdersReq) (*order.ListOrdersResp, error) {
	// todo: add your logic here and delete this line

	return &order.ListOrdersResp{}, nil
}
