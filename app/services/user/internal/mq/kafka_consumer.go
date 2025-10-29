package mq

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"NatsumeAI/app/services/agent/agent"
	"NatsumeAI/app/services/user/internal/svc"

	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-zero/core/logx"
)

// StartMerchantReviewConsumer starts a blocking Kafka consumer loop
// that reads merchant application events, invokes agent sync review,
// and updates the application status.
func StartMerchantReviewConsumer(ctx context.Context, sc *svc.ServiceContext) error {
    if len(sc.Config.KafkaConf.Broker) == 0 || sc.Config.KafkaConf.MerchantReviewTopic == "" || sc.Config.KafkaConf.Group == "" {
        return nil
    }
    r := kafka.NewReader(kafka.ReaderConfig{
        Brokers:     sc.Config.KafkaConf.Broker,
        GroupID:     sc.Config.KafkaConf.Group,
        Topic:       sc.Config.KafkaConf.MerchantReviewTopic,
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
        var evt MerchantReviewEvent
        if err := json.Unmarshal(m.Value, &evt); err == nil {
            handleMerchantReviewEvent(ctx, sc, evt)
        }
        _ = r.CommitMessages(ctx, m)
    }
}

func handleMerchantReviewEvent(ctx context.Context, sc *svc.ServiceContext, evt MerchantReviewEvent) {
    req := &agent.ReviewMerchantReq{
        ApplicationId: evt.ApplicationID,
        UserId:        evt.UserID,
        Application: &agent.MerchantApplicationInput{
            ShopName:     evt.Application.ShopName,
            ContactName:  evt.Application.ContactName,
            ContactPhone: evt.Application.ContactPhone,
            Address:      evt.Application.Address,
            Description:  evt.Application.Description,
        },
    }
    res, err := sc.AgentRpc.ReviewMerchant(ctx, req)
    if err != nil || res == nil {
        logx.WithContext(ctx).Errorw("agent review failed", logx.Field("err", err))
        updateMerchant(sc, evt.ApplicationID, "ESCALATED", "系统异常，转人工复核")
        return
    }
    switch res.GetDecision() {
    case agent.ReviewDecision_REVIEW_DECISION_APPROVE:
        updateMerchant(sc, evt.ApplicationID, "APPROVED", res.GetReason())
    case agent.ReviewDecision_REVIEW_DECISION_REJECT:
        updateMerchant(sc, evt.ApplicationID, "REJECTED", res.GetReason())
    default:
        updateMerchant(sc, evt.ApplicationID, "ESCALATED", res.GetReason())
    }
}

func updateMerchant(sc *svc.ServiceContext, appID int64, status, reason string) {
    rec, err := sc.MerchantsModel.FindOne(context.Background(), appID)
    if err != nil {
        logx.Errorw("find merchant application failed", logx.Field("err", err), logx.Field("application_id", appID))
        return
    }
    // Skip update if no state change to avoid unnecessary cache invalidation
    if rec.Status == status && rec.RejectReason == reason {
        return
    }
    rec.Status = status
    rec.RejectReason = reason

    if err := sc.MerchantsModel.Update(context.Background(), rec); err != nil {
        logx.Errorw("update merchant application failed", logx.Field("err", err), logx.Field("application_id", appID))
    }
    if status == "APPROVED" {
        sc.Casbin.AddRoleForUser(strconv.FormatInt(int64(rec.UserId), 10), "merchant")
    }
}
