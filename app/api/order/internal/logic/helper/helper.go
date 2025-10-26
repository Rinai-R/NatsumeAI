package helper

import (
    "NatsumeAI/app/api/order/internal/types"
    ordersrv "NatsumeAI/app/services/order/orderservice"
)

func ToOrderItemSnapshot(src *ordersrv.OrderItemSnapshot) types.OrderItemSnapshot {
    if src == nil {
        return types.OrderItemSnapshot{}
    }
    return types.OrderItemSnapshot{
        Title:       src.Title,
        Cover_image: src.CoverImage,
        Attributes:  src.Attributes,
    }
}

func ToOrderItem(src *ordersrv.OrderItem) types.OrderItem {
    if src == nil {
        return types.OrderItem{}
    }
    return types.OrderItem{
        Product_id:  src.ProductId,
        Quantity:    src.Quantity,
        Price_cents: src.PriceCents,
        Snapshot:    ToOrderItemSnapshot(src.Snapshot),
    }
}

func ToOrderInfo(src *ordersrv.OrderInfo) types.OrderInfo {
    if src == nil {
        return types.OrderInfo{}
    }
    items := make([]types.OrderItem, 0, len(src.Items))
    for _, it := range src.Items {
        items = append(items, ToOrderItem(it))
    }
    return types.OrderInfo{
        Order_id:         src.OrderId,
        Preorder_id:      src.PreorderId,
        User_id:          src.UserId,
        Status:           int32(src.Status),
        Total_amount:     src.TotalAmount,
        Pay_amount:       src.PayAmount,
        Created_at:       src.CreatedAt,
        Paid_at:          src.PaidAt,
        Cancelled_at:     src.CancelledAt,
        Items:            items,
        Payment_method:   src.PaymentMethod,
        Address_snapshot: src.AddressSnapshot,
    }
}

func ToOrderInfos(list []*ordersrv.OrderInfo) []types.OrderInfo {
    if len(list) == 0 {
        return nil
    }
    res := make([]types.OrderInfo, 0, len(list))
    for _, oi := range list {
        res = append(res, ToOrderInfo(oi))
    }
    return res
}

