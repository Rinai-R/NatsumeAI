package mq

import (
    "context"
    "database/sql"
    "encoding/json"
    "time"

    orderdal "NatsumeAI/app/dal/order"
    invpb "NatsumeAI/app/services/inventory/inventory"
    couponsvcpb "NatsumeAI/app/services/coupon/coupon"
    "NatsumeAI/app/services/order/internal/svc"

    "github.com/hibiken/asynq"
    "github.com/segmentio/kafka-go"
)

// StartCheckoutConsumer starts a blocking Kafka consumer loop for checkout events.
// It performs pre-freeze and inserts preorder item, then schedules a delayed cancel task.
func StartCheckoutConsumer(ctx context.Context, sc *svc.ServiceContext) error {
    if len(sc.Config.KafkaConf.Broker) == 0 || sc.Config.KafkaConf.PreOrderTopic == "" || sc.Config.KafkaConf.Group == "" {
        return nil
    }
    r := kafka.NewReader(kafka.ReaderConfig{
        Brokers:     sc.Config.KafkaConf.Broker,
        GroupID:     sc.Config.KafkaConf.Group,
        Topic:       sc.Config.KafkaConf.PreOrderTopic,
        MinBytes:    1,
        MaxBytes:    10 << 20,
        StartOffset: kafka.FirstOffset,
    })
    defer r.Close()

    for {
        m, err := r.FetchMessage(ctx)
        if err != nil {
            if ctx.Err() != nil { return nil }
            continue
        }
        var evt CheckoutEvent
        if err := json.Unmarshal(m.Value, &evt); err == nil {
            _ = handleCheckout(ctx, sc, evt)
        }
        _ = r.CommitMessages(ctx, m)
    }
}

// handleCheckout continues the flow after TryGetToken: pre-freeze, insert item, schedule cancel.
func handleCheckout(c context.Context, s *svc.ServiceContext, e CheckoutEvent) error {
    preorderID := e.PreorderId
    // pre-freeze inventory
    if _, err := s.Inventory.DecreasePreInventory(c, &invpb.InventoryReq{OrderId: preorderID, PreorderId: preorderID, Item: &invpb.Item{ProductId: e.ProductId, Quantity: e.Quantity}}); err != nil {
        // if pre-freeze fails, return token and delete preorder
        if po, ferr := s.Preorder.FindOne(c, preorderID); ferr == nil && po.CouponId > 0 && s.Coupon != nil {
            _, _ = s.Coupon.ReleaseCoupon(c, &couponsvcpb.ReleaseCouponReq{UserId: po.UserId, CouponId: po.CouponId, OrderId: preorderID})
        }
        _, _ = s.Inventory.ReturnToken(c, &invpb.ReturnTokenReq{PreorderId: preorderID, Item: []*invpb.Item{{ProductId: e.ProductId, Quantity: e.Quantity}}})
        _ = s.Preorder.Delete(c, preorderID)
        return err
    }

    // build snapshot JSON
    var snapStr sql.NullString
    if e.Snapshot != nil {
        if b, err := json.Marshal(e.Snapshot); err == nil {
            snapStr = sql.NullString{String: string(b), Valid: true}
        }
    }
    // insert preorder item
    if _, err := s.PreItm.Insert(c, &orderdal.OrderPreorderItems{PreorderId: preorderID, ProductId: e.ProductId, Quantity: e.Quantity, PriceCents: e.PriceCents, Snapshot: snapStr}); err != nil {
        _, _ = s.Inventory.ReturnPreInventory(c, &invpb.InventoryReq{OrderId: preorderID, PreorderId: preorderID, Item: &invpb.Item{ProductId: e.ProductId, Quantity: e.Quantity}})
        _, _ = s.Inventory.ReturnToken(c, &invpb.ReturnTokenReq{PreorderId: preorderID, Item: []*invpb.Item{{ProductId: e.ProductId, Quantity: e.Quantity}}})
        if po, ferr := s.Preorder.FindOne(c, preorderID); ferr == nil && po.CouponId > 0 && s.Coupon != nil {
            _, _ = s.Coupon.ReleaseCoupon(c, &couponsvcpb.ReleaseCouponReq{UserId: po.UserId, CouponId: po.CouponId, OrderId: preorderID})
        }
        _ = s.Preorder.Delete(c, preorderID)
        return err
    }

    // enqueue delayed cancel task
    if s.AsynqClient != nil {
        // compute delay based on preorder expire time
        delay := 30 * time.Minute
        if po, err := s.Preorder.FindOne(c, preorderID); err == nil {
            if d := time.Until(po.ExpireAt); d > 0 { delay = d } else { delay = time.Second * 1 }
        }
        payload, _ := json.Marshal(CancelTaskPayload{PreorderId: preorderID, UserId: e.UserId})
        task := asynq.NewTask(TaskCancelPreorder, payload)
        _, _ = s.AsynqClient.Enqueue(task, asynq.ProcessIn(delay), asynq.Queue("default"))
    }
    return nil
}

// NewAsynqMux registers handlers for delayed tasks.
func NewAsynqMux(sc *svc.ServiceContext) *asynq.ServeMux {
    mux := asynq.NewServeMux()
    mux.HandleFunc(TaskCancelPreorder, func(ctx context.Context, t *asynq.Task) error {
        var p CancelTaskPayload
        if err := json.Unmarshal(t.Payload(), &p); err != nil { return err }
        // check if order already created/paid, skip
        if _, err := sc.Orders.FindOneByPreorderId(ctx, p.PreorderId); err == nil {
            return nil // order exists, do nothing
        }
        // rollback resources and mark preorder cancelled if still pending
        // get item for product/qty
        var itemProductId int64
        var itemQty int64
        if rows, err := sc.PreItm.ListByPreorder(ctx, p.PreorderId); err == nil && len(rows) > 0 {
            itemProductId, itemQty = rows[0].ProductId, rows[0].Quantity
        }
        // return pre-inventory
        _, _ = sc.Inventory.ReturnPreInventory(ctx, &invpb.InventoryReq{OrderId: p.PreorderId, PreorderId: p.PreorderId, Item: &invpb.Item{ProductId: itemProductId, Quantity: itemQty}})
        // return token
        _, _ = sc.Inventory.ReturnToken(ctx, &invpb.ReturnTokenReq{PreorderId: p.PreorderId, Item: []*invpb.Item{{ProductId: itemProductId, Quantity: itemQty}}})
        // release coupon if locked
        if po, err := sc.Preorder.FindOne(ctx, p.PreorderId); err == nil {
            if po.CouponId > 0 && sc.Coupon != nil {
                _, _ = sc.Coupon.ReleaseCoupon(ctx, &couponsvcpb.ReleaseCouponReq{UserId: po.UserId, CouponId: po.CouponId, OrderId: p.PreorderId})
            }
            // mark preorder cancelled
            po.Status = "CANCELLED"
            _ = sc.Preorder.Update(ctx, po)
        }
        return nil
    })
    return mux
}
