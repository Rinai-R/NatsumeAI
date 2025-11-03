package server

import (
	"context"

	"NatsumeAI/app/services/payment/internal/logic"
	"NatsumeAI/app/services/payment/internal/svc"
	paymentpb "NatsumeAI/app/services/payment/payment"
)

type PaymentServiceServer struct {
	svcCtx *svc.ServiceContext
	paymentpb.UnimplementedPaymentServiceServer
}

func NewPaymentServiceServer(svcCtx *svc.ServiceContext) *PaymentServiceServer {
	return &PaymentServiceServer{
		svcCtx: svcCtx,
	}
}

func (s *PaymentServiceServer) CreatePayment(ctx context.Context, in *paymentpb.CreatePaymentReq) (*paymentpb.CreatePaymentResp, error) {
	l := logic.NewCreatePaymentLogic(ctx, s.svcCtx)
	return l.CreatePayment(in)
}

func (s *PaymentServiceServer) GetPayment(ctx context.Context, in *paymentpb.GetPaymentReq) (*paymentpb.GetPaymentResp, error) {
	l := logic.NewGetPaymentLogic(ctx, s.svcCtx)
	return l.GetPayment(in)
}
