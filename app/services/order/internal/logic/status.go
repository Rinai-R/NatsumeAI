package logic

import (
    "strings"
    "NatsumeAI/app/services/order/order"
)

// toProtoStatus maps internal order status string to protobuf enum.
func toProtoStatus(s string) order.OrderStatus {
    switch strings.ToUpper(s) {
    case "PENDING_PAYMENT", "PENDING":
        return order.OrderStatus_ORDER_STATUS_PENDING
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

