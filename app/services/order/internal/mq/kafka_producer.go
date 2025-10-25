package mq

import (
    "context"
    "encoding/json"
    "time"

    "NatsumeAI/app/services/order/internal/svc"
    "github.com/segmentio/kafka-go"
)

// PublishCheckoutEvent sends the checkout event to Kafka.
func PublishCheckoutEvent(sc *svc.ServiceContext, evt CheckoutEvent) error {
    if len(sc.Config.KafkaConf.Broker) == 0 || sc.Config.KafkaConf.PreOrderTopic == "" {
        return nil
    }
    body, err := json.Marshal(evt)
    if err != nil { return err }
    w := &kafka.Writer{
        Addr:         kafka.TCP(sc.Config.KafkaConf.Broker...),
        Topic:        sc.Config.KafkaConf.PreOrderTopic,
        RequiredAcks: kafka.RequireOne,
        Balancer:     &kafka.LeastBytes{},
        BatchTimeout: 10 * time.Millisecond,
    }
    defer w.Close()
    msg := kafka.Message{Key: nil, Value: body}
    return w.WriteMessages(context.Background(), msg)
}
