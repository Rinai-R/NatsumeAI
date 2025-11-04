package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"NatsumeAI/app/services/user/internal/review"
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
		fmt.Println("消息来了")
		m, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
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
	if sc.ChatModel == nil {
		logx.WithContext(ctx).Error("chat model not initialized, escalate to manual review")
		updateMerchant(sc, evt.ApplicationID, "ESCALATED", "系统异常，转人工复核")
		return
	}

	app := evt.Application
	summary := fmt.Sprintf("申请ID:%d\n用户ID:%d\n店铺名称:%s\n联系人:%s\n电话:%s\n地址:%s\n申请理由:%s",
		evt.ApplicationID, evt.UserID,
		safeStr(app.ShopName), safeStr(app.ContactName), safeStr(app.ContactPhone), safeStr(app.Address), safeStr(app.Description))

	out := review.NewReviewer(ctx, sc).Review(summary)
	if out == nil {
		logx.WithContext(ctx).Error("review model returned empty response")
		updateMerchant(sc, evt.ApplicationID, "ESCALATED", "系统异常，转人工复核")
		return
	}

	if out.Ok {
		updateMerchant(sc, evt.ApplicationID, "APPROVED", out.Reason)
		return
	}
	updateMerchant(sc, evt.ApplicationID, "REJECTED", out.Reason)
}

func updateMerchant(sc *svc.ServiceContext, appID int64, status, reason string) {
	rec, err := sc.MerchantsModel.FindOne(context.Background(), appID)
	if err != nil {
		logx.Errorw("find merchant application failed", logx.Field("err", err), logx.Field("application_id", appID))
		return
	}
	// 直接跳过
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

func safeStr(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
