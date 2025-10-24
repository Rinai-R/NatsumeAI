package logic

import (
	"context"

	"NatsumeAI/app/services/order/internal/svc"
	"NatsumeAI/app/services/order/order"

	"github.com/zeromicro/go-zero/core/logx"
)

type ConfirmPaymentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewConfirmPaymentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfirmPaymentLogic {
	return &ConfirmPaymentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 支付确认（支付成功）
func (l *ConfirmPaymentLogic) ConfirmPayment(in *order.ConfirmPaymentReq) (*order.ConfirmPaymentResp, error) {
	// todo: add your logic here and delete this line

	return &order.ConfirmPaymentResp{}, nil
}
