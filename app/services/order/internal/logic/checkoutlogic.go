package logic

import (
	"context"

	"NatsumeAI/app/services/order/internal/svc"
	"NatsumeAI/app/services/order/order"

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

// Checkout（创建订单草稿，发令牌）
func (l *CheckoutLogic) Checkout(in *order.CheckoutReq) (*order.CheckoutResp, error) {
	// todo: add your logic here and delete this line

	return &order.CheckoutResp{}, nil
}
