package mq

import (
    "context"
    "encoding/json"
    "net"
    "strconv"
    "strings"
    "time"

    "NatsumeAI/app/services/order/internal/svc"

    "github.com/segmentio/kafka-go"
    "github.com/zeromicro/go-zero/core/logx"
)

// 发送 checkout 消息
func PublishCheckoutEvent(sc *svc.ServiceContext, evt CheckoutEvent) error {
    if len(sc.Config.KafkaConf.Broker) == 0 || sc.Config.KafkaConf.PreOrderTopic == "" {
        return nil
    }
    body, err := json.Marshal(evt)
    if err != nil { return err }
    // Use shared writer if available to reduce latency; fallback to ephemeral writer.
    w := sc.KafkaWriter
    var closer func() error
    if w == nil {
        ww := &kafka.Writer{
            Addr:         kafka.TCP(sc.Config.KafkaConf.Broker...),
            Topic:        sc.Config.KafkaConf.PreOrderTopic,
            RequiredAcks: kafka.RequireOne,
            Balancer:     &kafka.LeastBytes{},
            BatchTimeout: 5 * time.Millisecond,
            AllowAutoTopicCreation: true,
        }
        w = ww
        closer = ww.Close
    }
    msg := kafka.Message{
        Key:   nil,
        Value: body,
    }
    if err := w.WriteMessages(context.Background(), msg); err != nil {
        // 如果是未知主题错误，尝试创建后重试一次
        if isUnknownTopicErr(err) {
            ensureTopic(sc, sc.Config.KafkaConf.PreOrderTopic, 1)
            time.Sleep(200 * time.Millisecond)
            return w.WriteMessages(context.Background(), msg)
        }
        return err
    }
    if closer != nil { _ = closer() }
    return nil
}

// isUnknownTopicErr performs a best-effort check on broker error text.
func isUnknownTopicErr(err error) bool {
    if err == nil { return false }
    s := strings.ToLower(err.Error())
    return strings.Contains(s, "unknown topic") || strings.Contains(s, "invalid topic")
}

// ensureTopic tries to create the topic via controller; logs and ignores errors.
func ensureTopic(sc *svc.ServiceContext, topic string, partitions int) {
    if len(sc.Config.KafkaConf.Broker) == 0 || topic == "" {
        return
    }
    brk := sc.Config.KafkaConf.Broker[0]
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()
    conn, err := kafka.DialContext(ctx, "tcp", brk)
    if err != nil {
        logx.WithContext(ctx).Infof("kafka publish ensure topic: dial broker failed: %v", err)
        return
    }
    defer conn.Close()
    controller, err := conn.Controller()
    if err != nil {
        logx.WithContext(ctx).Infof("kafka publish ensure topic: controller failed: %v", err)
        return
    }
    addr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
    cctx, ccancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer ccancel()
    cconn, err := kafka.DialContext(cctx, "tcp", addr)
    if err != nil {
        logx.WithContext(cctx).Infof("kafka publish ensure topic: dial controller failed: %v", err)
        return
    }
    defer cconn.Close()
    if partitions <= 0 { partitions = 1 }
    cfg := kafka.TopicConfig{Topic: topic, NumPartitions: partitions, ReplicationFactor: 1}
    if err := cconn.CreateTopics(cfg); err != nil {
        logx.WithContext(cctx).Infof("kafka publish ensure topic: create result: %v", err)
    } else {
        logx.WithContext(cctx).Infof("kafka publish ensure topic: created %s", topic)
    }
}
