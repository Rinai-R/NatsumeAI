package mq

import (
    "context"
    "encoding/json"
    "time"

    "NatsumeAI/app/services/user/internal/svc"

    "github.com/segmentio/kafka-go"
)

// PublishMerchantReviewEvent sends a merchant review event to Kafka.
// Uses the shared writer in ServiceContext when available, else creates a
// short-lived writer to publish one message.
func PublishMerchantReviewEvent(sc *svc.ServiceContext, evt MerchantReviewEvent) error {
    if len(sc.Config.KafkaConf.Broker) == 0 || sc.Config.KafkaConf.MerchantReviewTopic == "" {
        return nil
    }
    body, err := json.Marshal(evt)
    if err != nil { return err }

    w := sc.KafkaWriter
    var closer func() error
    if w == nil {
        ww := &kafka.Writer{
            Addr:         kafka.TCP(sc.Config.KafkaConf.Broker...),
            Topic:        sc.Config.KafkaConf.MerchantReviewTopic,
            RequiredAcks: kafka.RequireOne,
            Balancer:     &kafka.LeastBytes{},
            BatchTimeout: 5 * time.Millisecond,
            AllowAutoTopicCreation: true,
        }
        w = ww
        closer = ww.Close
    }
    msg := kafka.Message{Value: body}
    if err := w.WriteMessages(context.Background(), msg); err != nil {
        return err
    }
    if closer != nil { _ = closer() }
    return nil
}

