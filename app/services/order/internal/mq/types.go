package mq

// CheckoutEvent defines the payload sent to Kafka to trigger preorder creation and inventory pre-freeze.
type CheckoutEvent struct {
    PreorderId int64  `json:"preorder_id"`
    UserId     int64  `json:"user_id"`
    ProductId  int64  `json:"product_id"`
    Quantity   int64  `json:"quantity"`
    PriceCents int64  `json:"price_cents"`
    Snapshot   *CheckoutSnapshot `json:"snapshot,omitempty"`
}

// Asynq task type for checkout events
const TaskCheckout = "order:checkout"
const TaskCancelPreorder = "order:cancel_if_unpaid"

// CheckoutSnapshot carries product info to enrich preorder item snapshot.
type CheckoutSnapshot struct {
    Title      string `json:"title"`
    CoverImage string `json:"cover_image"`
    Attributes string `json:"attributes"`
}

// CancelTaskPayload represents payload for delayed cancel
type CancelTaskPayload struct {
    PreorderId int64 `json:"preorder_id"`
    UserId     int64 `json:"user_id"`
}
