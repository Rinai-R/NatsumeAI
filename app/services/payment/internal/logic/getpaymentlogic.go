package logic

import (
	"context"

	paymentdal "NatsumeAI/app/dal/payment"
	"NatsumeAI/app/services/payment/internal/svc"
	paymentpb "NatsumeAI/app/services/payment/payment"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPaymentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPaymentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPaymentLogic {
	return &GetPaymentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetPaymentLogic) GetPayment(in *paymentpb.GetPaymentReq) (*paymentpb.GetPaymentResp, error) {
	resp := &paymentpb.GetPaymentResp{}
	if in == nil || (in.PaymentId <= 0 && in.PaymentNo == "" && in.OrderId <= 0) {
		resp.StatusCode = 400
		resp.StatusMsg = "invalid params"
		return resp, nil
	}

	var (
		po  *paymentdal.PaymentOrders
		err error
	)

	switch {
	case in.PaymentId > 0:
		po, err = l.svcCtx.PaymentOrders.FindOne(l.ctx, in.PaymentId)
	case in.PaymentNo != "":
		po, err = l.svcCtx.PaymentOrders.FindOneByPaymentNo(l.ctx, in.PaymentNo)
	default:
		po, err = l.svcCtx.PaymentOrders.FindOneByOrderId(l.ctx, in.OrderId)
	}

	if err != nil {
		if err == paymentdal.ErrNotFound {
			resp.StatusCode = 404
			resp.StatusMsg = "payment not found"
			return resp, nil
		}
		return nil, err
	}

	resp.StatusCode = 0
	resp.StatusMsg = "ok"
	resp.Payment = toPaymentInfo(po)
	return resp, nil
}
