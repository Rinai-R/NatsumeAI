package logic

import (
	"NatsumeAI/app/services/order/order"
	"strings"
)

// toProtoStatus maps internal order status string to protobuf enum.
func toProtoStatus(s string) order.OrderStatus {
	switch strings.ToUpper(s) {
	case "PENDING_PAYMENT", "PENDING":
		return order.OrderStatus_ORDER_STATUS_PENDING
	case "PAYING":
		return order.OrderStatus_ORDER_STATUS_PAYING
	case "PAID", "CONFIRMED":
		return order.OrderStatus_ORDER_STATUS_CONFIRMED
	case "CANCELLED", "CANCELED":
		return order.OrderStatus_ORDER_STATUS_CANCELLED
	case "COMPLETED", "DONE", "FINISHED":
		return order.OrderStatus_ORDER_STATUS_COMPLETED
	default:
		return order.OrderStatus_ORDER_STATUS_UNKNOWN
	}
}
