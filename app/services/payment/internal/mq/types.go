package mq

const TaskExpirePayment = "payment:expire_payment"

type ExpirePaymentPayload struct {
	PaymentId int64 `json:"payment_id"`
	OrderId   int64 `json:"order_id"`
	UserId    int64 `json:"user_id"`
}
