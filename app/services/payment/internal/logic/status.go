package logic

import (
	paymentdal "NatsumeAI/app/dal/payment"
	paymentpb "NatsumeAI/app/services/payment/payment"
)

func toProtoStatus(s string) paymentpb.PaymentStatus {
	switch s {
	case "INIT":
		return paymentpb.PaymentStatus_PAYMENT_STATUS_INIT
	case "PROCESSING":
		return paymentpb.PaymentStatus_PAYMENT_STATUS_PROCESSING
	case "SUCCESS":
		return paymentpb.PaymentStatus_PAYMENT_STATUS_SUCCESS
	case "FAILED":
		return paymentpb.PaymentStatus_PAYMENT_STATUS_FAILED
	case "CANCELLED":
		return paymentpb.PaymentStatus_PAYMENT_STATUS_CANCELLED
	case "EXPIRED":
		return paymentpb.PaymentStatus_PAYMENT_STATUS_EXPIRED
	default:
		return paymentpb.PaymentStatus_PAYMENT_STATUS_UNKNOWN
	}
}

func toPaymentInfo(po *paymentdal.PaymentOrders) *paymentpb.PaymentInfo {
	if po == nil {
		return nil
	}
	return &paymentpb.PaymentInfo{
		PaymentId: po.PaymentId,
		PaymentNo: po.PaymentNo,
		Status:    toProtoStatus(po.Status),
		OrderId:   po.OrderId,
		UserId:    po.UserId,
		Amount:    po.Amount,
		Currency:  po.Currency,
		Channel:   po.Channel,
		Credential: func() string {
			if po.ChannelPayload.Valid {
				return po.ChannelPayload.String
			}
			return ""
		}(),
		TimeoutAt: po.TimeoutAt.Unix(),
	}
}
