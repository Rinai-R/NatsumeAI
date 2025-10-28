package mq

// MerchantApplicationPayload captures the merchant application data sent in events.
type MerchantApplicationPayload struct {
    ShopName     string `json:"shop_name"`
    ContactName  string `json:"contact_name"`
    ContactPhone string `json:"contact_phone"`
    Address      string `json:"address"`
    Description  string `json:"description"`
}

// MerchantReviewEvent is published to trigger AI review.
// It contains the DB-generated application id and the normalized application payload.
type MerchantReviewEvent struct {
    ApplicationID int64                      `json:"application_id"`
    UserID        int64                      `json:"user_id"`
    Application   MerchantApplicationPayload `json:"application"`
}

// MerchantPublishPayload is the message body used for DTM commit-and-submit publish step.
// It does not include application_id because it is generated within the local DB txn.
type MerchantPublishPayload struct {
    UserID      int64                      `json:"user_id"`
    Application MerchantApplicationPayload `json:"application"`
}
