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

type ConfirmPaymentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewConfirmPaymentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfirmPaymentLogic {
	return &ConfirmPaymentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ConfirmPaymentLogic) ConfirmPayment(req *types.ConfirmPaymentRequest) (resp *types.ConfirmPaymentResponse, err error) {
    uid, _ := util.UserIdFromCtx(l.ctx)
    in := &orderservice.ConfirmPaymentReq{
        OrderId:       req.Order_id,
        UserId:        uid,
        PaymentMethod: req.Payment_method,
        PaymentToken:  req.Payment_token,
    }
    out, err := l.svcCtx.OrderRpc.ConfirmPayment(l.ctx, in)
    if err != nil {
        return nil, err
    }
    return &types.ConfirmPaymentResponse{
        Status_code: out.StatusCode,
        Status_msg:  out.StatusMsg,
        Status:      int32(out.Status),
    }, nil
}
