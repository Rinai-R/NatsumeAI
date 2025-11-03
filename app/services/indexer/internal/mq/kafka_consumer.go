package mq

import (
	"NatsumeAI/app/services/indexer/internal/svc"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-zero/core/logx"
)

func StartCanalProductConsumer(ctx context.Context, sc *svc.ServiceContext) error {
	if len(sc.Config.KafkaConf.Brokers) == 0 || sc.Config.KafkaConf.ProductsTopic == "" || sc.Config.KafkaConf.Group == "" {
		logx.Infow("skip product consumer, kafka config missing")
		return nil
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     sc.Config.KafkaConf.Brokers,
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
			if ctx.Err() != nil {
				return nil
			}
			logx.Errorw("fetch product message failed", logx.Field("err", err))
			continue
		}

		var evt CanalMessageProducts
		if err := json.Unmarshal(m.Value, &evt); err != nil {
			logx.Errorw("unmarshal product message failed", logx.Field("err", err))
		} else {
			handleCanalProductMessage(ctx, sc, evt)
		}

		if err := r.CommitMessages(ctx, m); err != nil {
			logx.Errorw("commit product message failed", logx.Field("err", err))
		}
	}
}

func handleCanalProductMessage(ctx context.Context, sc *svc.ServiceContext, message CanalMessageProducts) {
	if sc.ESClient == nil {
		logx.Infow("skip product message, elasticsearch client unavailable")
		return
	}

	if len(message.Data) == 0 {
		return
	}

	indexName := sc.ProductIndexName()
	eventType := strings.ToUpper(message.Type)

	for _, row := range message.Data {
		docID := strconv.FormatInt(row.ID, 10)
		switch eventType {
		case "DELETE":
			if err := deleteProductDocument(ctx, sc, indexName, docID); err != nil {
				logx.Errorw("delete product document failed", logx.Field("id", docID), logx.Field("err", err))
			}
		default:
			if err := upsertProductDocument(ctx, sc, indexName, docID, row, eventType); err != nil {
				logx.Errorw("upsert product document failed", logx.Field("id", docID), logx.Field("err", err))
			}
		}
	}
}

func StartCanalProductCategoryConsumer(ctx context.Context, sc *svc.ServiceContext) error {
	if len(sc.Config.KafkaConf.Brokers) == 0 || sc.Config.KafkaConf.ProductCategoryTopic == "" || sc.Config.KafkaConf.Group == "" {
		logx.Infow("skip product category consumer, kafka config missing")
		return nil
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     sc.Config.KafkaConf.Brokers,
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
			if ctx.Err() != nil {
				return nil
			}
			logx.Errorw("fetch category message failed", logx.Field("err", err))
			continue
		}

		var evt CanalProductCategoryMessage
		if err := json.Unmarshal(m.Value, &evt); err != nil {
			logx.Errorw("unmarshal category message failed", logx.Field("err", err))
		} else {
			handleCanalProductCategoryMessage(ctx, sc, evt)
		}

		if err := r.CommitMessages(ctx, m); err != nil {
			logx.Errorw("commit category message failed", logx.Field("err", err))
		}
	}
}

func handleCanalProductCategoryMessage(ctx context.Context, sc *svc.ServiceContext, message CanalProductCategoryMessage) {
	if sc.ESClient == nil {
		logx.Infow("skip category message, elasticsearch client unavailable")
		return
	}

	if len(message.Data) == 0 {
		return
	}

	indexName := sc.ProductIndexName()
	eventType := strings.ToUpper(message.Type)

	for _, row := range message.Data {
		if err := updateProductCategories(ctx, sc, indexName, row, eventType); err != nil {
			logx.Errorw("update product categories failed",
				logx.Field("product_id", row.ProductID),
				logx.Field("category", row.Category),
				logx.Field("err", err),
			)
		}
	}
}

func upsertProductDocument(ctx context.Context, sc *svc.ServiceContext, indexName, docID string, row ProductRow, eventType string) error {
	doc := map[string]any{
		"product_id":  row.ID,
		"merchant_id": row.MerchantID,
		"name":        row.Name,
		"description": row.Description,
		"picture":     row.Picture,
		"price":       row.Price,
	}

	if createdAt := normalizeProductTimestamp(row.CreatedAt); createdAt != "" {
		doc["created_at"] = createdAt
	}

	if updatedAt := normalizeProductTimestamp(row.UpdatedAt); updatedAt != "" {
		doc["updated_at"] = updatedAt
	}

	if sc.VectorIndexEnabled() {
		if embedding, err := buildProductEmbedding(ctx, sc, row); err != nil {
			logx.Errorw("compute product embedding failed", logx.Field("id", docID), logx.Field("err", err))
		} else if len(embedding) > 0 {
			doc["embedding"] = embedding
		}
	}

	if eventType == "INSERT" {
		doc["categories"] = []string{}
	}

	payload := map[string]any{
		"doc":           doc,
		"doc_as_upsert": true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode product document: %w", err)
	}

	res, err := sc.ESClient.Update(indexName, docID, bytes.NewReader(body), sc.ESClient.Update.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("es update call: %w", err)
	}
	defer res.Body.Close()

	respBody, _ := io.ReadAll(res.Body)
	if res.IsError() {
		return fmt.Errorf("es update status %s: %s", res.Status(), strings.TrimSpace(string(respBody)))
	}

	return nil
}

func deleteProductDocument(ctx context.Context, sc *svc.ServiceContext, indexName, docID string) error {
	res, err := sc.ESClient.Delete(indexName, docID, sc.ESClient.Delete.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("es delete call: %w", err)
	}
	defer res.Body.Close()

	respBody, _ := io.ReadAll(res.Body)
	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("es delete status %s: %s", res.Status(), strings.TrimSpace(string(respBody)))
	}

	return nil
}

func updateProductCategories(ctx context.Context, sc *svc.ServiceContext, indexName string, row ProductCategoryRow, eventType string) error {
	docID := strconv.FormatInt(row.ProductID, 10)

	categories, exists, err := fetchExistingCategories(ctx, sc, indexName, docID)
	if err != nil {
		return fmt.Errorf("fetch categories: %w", err)
	}

	if !exists && eventType == "DELETE" {
		return nil
	}

	switch eventType {
	case "DELETE":
		categories = removeString(categories, row.Category)
	default:
		categories = addString(categories, row.Category)
	}

	updateDoc := map[string]any{
		"categories":          categories,
		"category_updated_at": time.Now().UTC().Format(time.RFC3339),
	}
	if !exists {
		updateDoc["product_id"] = row.ProductID
	}

	payload := map[string]any{
		"doc":           updateDoc,
		"doc_as_upsert": true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode category update: %w", err)
	}

	res, err := sc.ESClient.Update(indexName, docID, bytes.NewReader(body), sc.ESClient.Update.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("es update call: %w", err)
	}
	defer res.Body.Close()

	respBody, _ := io.ReadAll(res.Body)
	if res.IsError() {
		return fmt.Errorf("es update status %s: %s", res.Status(), strings.TrimSpace(string(respBody)))
	}

	return nil
}

func fetchExistingCategories(ctx context.Context, sc *svc.ServiceContext, indexName, docID string) ([]string, bool, error) {
	res, err := sc.ESClient.Get(indexName, docID,
		sc.ESClient.Get.WithContext(ctx),
		sc.ESClient.Get.WithSourceIncludes("categories"))
	if err != nil {
		return nil, false, err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, false, nil
	}

	respBody, _ := io.ReadAll(res.Body)
	if res.IsError() {
		return nil, false, fmt.Errorf("es get status %s: %s", res.Status(), strings.TrimSpace(string(respBody)))
	}

	var payload struct {
		Source map[string]any `json:"_source"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, false, fmt.Errorf("decode get response: %w", err)
	}

	return normalizeStringSlice(payload.Source["categories"]), true, nil
}

func normalizeStringSlice(v any) []string {
	if v == nil {
		return []string{}
	}

	switch cast := v.(type) {
	case []string:
		out := make([]string, 0, len(cast))
		for _, item := range cast {
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(cast))
		for _, item := range cast {
			if str, ok := item.(string); ok && str != "" {
				out = append(out, str)
			}
		}
		return out
	default:
		return []string{}
	}
}

func removeString(slice []string, target string) []string {
	if target == "" || len(slice) == 0 {
		return slice
	}

	out := slice[:0]
	for _, item := range slice {
		if !strings.EqualFold(item, target) {
			out = append(out, item)
		}
	}
	return out
}

func addString(slice []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return slice
	}

	for _, item := range slice {
		if strings.EqualFold(item, value) {
			return slice
		}
	}

	return append(slice, value)
}

func buildProductEmbedding(ctx context.Context, sc *svc.ServiceContext, row ProductRow) ([]float64, error) {
	if sc.Embedder == nil {
		return nil, nil
	}

	textParts := []string{
		row.Name,
		row.Description,
		row.Picture,
		strconv.FormatInt(row.Price, 10),
	}
	text := strings.TrimSpace(strings.Join(textParts, " "))
	if text == "" {
		text = strconv.FormatInt(row.ID, 10)
	}

	embeds, err := sc.Embedder.EmbedStrings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeds) == 0 || len(embeds[0]) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}
	if expected := sc.EmbeddingDimension(); expected > 0 && len(embeds[0]) != expected {
		return nil, fmt.Errorf("embedding dimension mismatch, expect %d got %d", expected, len(embeds[0]))
	}
	return embeds[0], nil
}

func normalizeProductTimestamp(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
	}

	for _, layout := range layouts {
		var (
			parsed time.Time
			err    error
		)

		switch layout {
		case "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05":
			parsed, err = time.ParseInLocation(layout, raw, time.Local)
		default:
			parsed, err = time.Parse(layout, raw)
		}

		if err == nil {
			return parsed.Format(time.RFC3339Nano)
		}
	}

	if !strings.Contains(raw, "T") && strings.Contains(raw, " ") {
		return strings.Replace(raw, " ", "T", 1)
	}

	return raw
}

func normalizeProductTimestamp(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
	}

	for _, layout := range layouts {
		var (
			parsed time.Time
			err    error
		)

		switch layout {
		case "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05":
			parsed, err = time.ParseInLocation(layout, raw, time.Local)
		default:
			parsed, err = time.Parse(layout, raw)
		}

		if err == nil {
			return parsed.Format(time.RFC3339Nano)
		}
	}

	if !strings.Contains(raw, "T") && strings.Contains(raw, " ") {
		return strings.Replace(raw, " ", "T", 1)
	}

	return raw
}
