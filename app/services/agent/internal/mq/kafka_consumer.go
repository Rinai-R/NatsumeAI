package mq

import (
	"NatsumeAI/app/services/agent/internal/svc"
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
)


func StartCanalProductConsumer(ctx context.Context, sc *svc.ServiceContext) error {
    if len(sc.Config.KafkaConf.Broker) == 0 || sc.Config.KafkaConf.ProductsTopic == "" || sc.Config.KafkaConf.Group == "" {
        return nil
    }
    r := kafka.NewReader(kafka.ReaderConfig{
        Brokers:     sc.Config.KafkaConf.Broker,
        GroupID:     sc.Config.KafkaConf.Group,
        Topic:       sc.Config.KafkaConf.ProductsTopic,
        MinBytes:    1,
        MaxBytes:    10 << 20,
        MaxWait:     50 * time.Millisecond,
        StartOffset: kafka.FirstOffset,
    })
    defer r.Close()

    for {
        m, err := r.FetchMessage(ctx)
        if err != nil {
            if ctx.Err() != nil { return nil }
            continue
        }
        var evt CanalMessageProducts
        if err := json.Unmarshal(m.Value, &evt); err == nil {
            handleCanalProductMessage(ctx, sc, evt)
        }
        _ = r.CommitMessages(ctx, m)
    }
}

func handleCanalProductMessage(ctx context.Context, sc *svc.ServiceContext, message CanalMessageProducts) {

}


func StartCanalProductCategoryConsumer(ctx context.Context, sc *svc.ServiceContext) error {
    if len(sc.Config.KafkaConf.Broker) == 0 || sc.Config.KafkaConf.ProductCategoryTopic == "" || sc.Config.KafkaConf.Group == "" {
        return nil
    }
    r := kafka.NewReader(kafka.ReaderConfig{
        Brokers:     sc.Config.KafkaConf.Broker,
        GroupID:     sc.Config.KafkaConf.Group,
        Topic:       sc.Config.KafkaConf.ProductCategoryTopic,
        MinBytes:    1,
        MaxBytes:    10 << 20,
        MaxWait:     50 * time.Millisecond,
        StartOffset: kafka.FirstOffset,
    })
    defer r.Close()

    for {
        m, err := r.FetchMessage(ctx)
        if err != nil {
            if ctx.Err() != nil { return nil }
            continue
        }
        var evt CanalProductCategoryMessage
        if err := json.Unmarshal(m.Value, &evt); err == nil {
            handleCanalProductCategoryMessage(ctx, sc, evt)
        }
        _ = r.CommitMessages(ctx, m)
    }
}

func handleCanalProductCategoryMessage(ctx context.Context, sc *svc.ServiceContext, message CanalProductCategoryMessage) {

}