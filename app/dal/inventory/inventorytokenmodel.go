package inventory

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	defaultTicketTTL         = 300 * time.Second
	tokenEpochKeyPattern     = "inv:%d:epoch"
	tokenThresholdKeyPattern = "inv:%d:threshold:%d"
	tokenIssuedKeyPattern    = "inv:%d:issued:%d"
	admissionTicketPattern   = "adm:%s"
)

//go:embed try_token.lua
var tryTokenScript string

//go:embed return_token.lua
var returnTokenScript string

type (
	// TokenItem describes the minimal information required to acquire or release tokens for an SKU.
	TokenItem struct {
		SKU      int64 `json:"sku"`
		Quantity int64 `json:"qty"`
		Epoch    int64 `json:"epoch"`
	}

	// TokenTicket captures the Redis admission ticket for a preorder.
	TokenTicket struct {
		PreorderID string      `json:"preorder_id"`
		IssuedAt   int64       `json:"issued_at"`
		Items      []TokenItem `json:"items"`
	}

	InventoryTokenModel interface {
		SyncTokenSnapshots(ctx context.Context, skus []int64) error
		TryGetToken(ctx context.Context, preorderID int64, items []TokenItem) error
		CheckToken(ctx context.Context, preorderID int64, consume bool) (*TokenTicket, error)
		ReturnToken(ctx context.Context, preorderID int64, items []TokenItem) (restored int64, skipped int64, err error)
	}

	defaultInventoryTokenModel struct {
		redis     *redis.Redis
		inventory InventoryModel
		trySha    string
		returnSha string
		mu        sync.Mutex
	}
)

// TokenError wraps status codes returned by lua scripts so callers can branch on error.Code.
type TokenError struct {
	code    string
	details []string
}

func (e *TokenError) Error() string {
	if len(e.details) == 0 {
		return fmt.Sprintf("inventory token error: %s", e.code)
	}
	return fmt.Sprintf("inventory token error: %s (%s)", e.code, strings.Join(e.details, ","))
}

// Code returns the machine-readable lua status code.
func (e *TokenError) Code() string {
	return e.code
}

// Details returns optional information returned by lua.
func (e *TokenError) Details() []string {
	return append([]string(nil), e.details...)
}

// Is allows errors.Is to work with the same code value.
func (e *TokenError) Is(target error) bool {
	other, ok := target.(*TokenError)
	if !ok {
		return false
	}
	return e.code == other.code
}

func newTokenError(code string, details ...string) *TokenError {
	return &TokenError{
		code:    code,
		details: append([]string(nil), details...),
	}
}

func NewInventoryTokenModel(r *redis.Redis, inventory InventoryModel) InventoryTokenModel {
	return &defaultInventoryTokenModel{
		redis:     r,
		inventory: inventory,
	}
}

func (m *defaultInventoryTokenModel) SyncTokenSnapshots(ctx context.Context, skus []int64) error {
	if len(skus) == 0 {
		return nil
	}

	seen := make(map[int64]struct{}, len(skus))
	for _, sku := range skus {
		if sku <= 0 {
			return newTokenError("INVALID_SKU", fmt.Sprintf("%d", sku))
		}
		if _, ok := seen[sku]; ok {
			continue
		}
		seen[sku] = struct{}{}
		if err := m.ensureSnapshotForSKU(ctx, sku); err != nil {
			return err
		}
	}

	return nil
}

func (m *defaultInventoryTokenModel) TryGetToken(ctx context.Context, preorderID int64, items []TokenItem) error {
	if preorderID <= 0 {
		return newTokenError("INVALID_PREORDER", fmt.Sprintf("%d", preorderID))
	}

	if err := m.ensureSnapshots(ctx, items); err != nil {
		return err
	}

	resolved, err := m.resolveEpochs(ctx, items)
	if err != nil {
		return err
	}

	normalized, err := normalizeItems(resolved)
	if err != nil {
		return err
	}

	if len(normalized) == 0 {
		return newTokenError("INVALID_ITEM_COUNT")
	}

	preorderStr := strconv.FormatInt(preorderID, 10)
	ticket := TokenTicket{
		PreorderID: preorderStr,
		IssuedAt:   time.Now().Unix(),
		Items:      normalized,
	}

	payload, err := json.Marshal(ticket)
	if err != nil {
		return fmt.Errorf("marshal ticket: %w", err)
	}

	keys := make([]string, 0, len(normalized)*3)
	for _, item := range normalized {
		keys = append(keys,
			fmt.Sprintf(tokenEpochKeyPattern, item.SKU),
			fmt.Sprintf(tokenThresholdKeyPattern, item.SKU, item.Epoch),
			fmt.Sprintf(tokenIssuedKeyPattern, item.SKU, item.Epoch),
		)
	}

	args := []any{
		len(normalized),
		preorderStr,
		int(defaultTicketTTL / time.Second),
		string(payload),
	}

	result, err := m.evalScript(ctx, &m.trySha, tryTokenScript, keys, args...)
	if err != nil {
		return err
	}

	status, details, parseErr := decodeLuaResult(result)
	if parseErr != nil {
		return parseErr
	}

	if status == "OK" {
		return nil
	}

	return newTokenError(status, details...)
}

func (m *defaultInventoryTokenModel) CheckToken(ctx context.Context, preorderID int64, consume bool) (*TokenTicket, error) {
	if preorderID <= 0 {
		return nil, newTokenError("INVALID_PREORDER", fmt.Sprintf("%d", preorderID))
	}

	key := AdmissionTicketKey(strconv.FormatInt(preorderID, 10))

	var (
		payload string
		err     error
	)

	if consume {
		payload, err = m.redis.GetDelCtx(ctx, key)
	} else {
		payload, err = m.redis.GetCtx(ctx, key)
	}
	if err != nil {
		return nil, err
	}
	if payload == "" {
		return nil, newTokenError("TICKET_NOT_FOUND", key)
	}

	ticket, err := decodeTicket(payload)
	if err != nil {
		return nil, err
	}

	if ticket.PreorderID == "" {
		ticket.PreorderID = strconv.FormatInt(preorderID, 10)
	}

	return ticket, nil
}

func (m *defaultInventoryTokenModel) ReturnToken(ctx context.Context, preorderID int64, items []TokenItem) (int64, int64, error) {
	if len(items) == 0 {
		return 0, 0, newTokenError("INVALID_ITEM_COUNT")
	}

	normalized, err := normalizeItems(items)
	if err != nil {
		return 0, 0, err
	}

	ticket := TokenTicket{
		PreorderID: strconv.FormatInt(preorderID, 10),
		IssuedAt:   time.Now().Unix(),
		Items:      normalized,
	}

	payload, err := json.Marshal(ticket)
	if err != nil {
		return 0, 0, fmt.Errorf("marshal ticket: %w", err)
	}

	keys := make([]string, 0, len(normalized)*3)
	for _, item := range normalized {
		keys = append(keys,
			fmt.Sprintf(tokenEpochKeyPattern, item.SKU),
			fmt.Sprintf(tokenThresholdKeyPattern, item.SKU, item.Epoch),
			fmt.Sprintf(tokenIssuedKeyPattern, item.SKU, item.Epoch),
		)
	}

	args := []any{
		len(normalized),
		string(payload),
		ticket.PreorderID,
	}

	result, err := m.evalScript(ctx, &m.returnSha, returnTokenScript, keys, args...)
	if err != nil {
		return 0, 0, err
	}

	status, details, parseErr := decodeLuaResult(result)
	if parseErr != nil {
		return 0, 0, parseErr
	}

	if status != "OK" {
		return 0, 0, newTokenError(status, details...)
	}

	var restored, skipped int64
	if len(details) > 0 {
		if v, convErr := strconv.ParseInt(details[0], 10, 64); convErr == nil {
			restored = v
		}
	}
	if len(details) > 1 {
		if v, convErr := strconv.ParseInt(details[1], 10, 64); convErr == nil {
			skipped = v
		}
	}
	return restored, skipped, nil
}

func (m *defaultInventoryTokenModel) evalScript(ctx context.Context, shaRef *string, script string, keys []string, args ...any) (any, error) {
	m.mu.Lock()
	sha := *shaRef
	m.mu.Unlock()

	if sha == "" {
		if err := m.loadScript(ctx, shaRef, script); err != nil {
			return nil, err
		}
		m.mu.Lock()
		sha = *shaRef
		m.mu.Unlock()
	}

	result, err := m.redis.EvalShaCtx(ctx, sha, keys, args...)
	if err != nil && strings.Contains(err.Error(), "NOSCRIPT") {
		logx.WithContext(ctx).Slowf("redis script hash lost (%s), reloading", sha)
		if loadErr := m.loadScript(ctx, shaRef, script); loadErr != nil {
			return nil, loadErr
		}
		m.mu.Lock()
		sha = *shaRef
		m.mu.Unlock()
		result, err = m.redis.EvalShaCtx(ctx, sha, keys, args...)
	}
	return result, err
}

func (m *defaultInventoryTokenModel) loadScript(ctx context.Context, shaRef *string, script string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// double check in case another goroutine already loaded it.
	if *shaRef != "" {
		return nil
	}

	hash, err := m.redis.ScriptLoadCtx(ctx, script)
	if err != nil {
		return err
	}
	*shaRef = hash
	return nil
}

func normalizeItems(items []TokenItem) ([]TokenItem, error) {
	type itemKey struct {
		sku   int64
		epoch int64
	}

	index := make(map[itemKey]int64, len(items))
	order := make([]itemKey, 0, len(items))

	for _, item := range items {
		if item.SKU <= 0 {
			return nil, newTokenError("INVALID_SKU", fmt.Sprintf("%d", item.SKU))
		}
		if item.Epoch <= 0 {
			return nil, newTokenError("INVALID_EPOCH", fmt.Sprintf("%d", item.Epoch))
		}
		if item.Quantity <= 0 {
			return nil, newTokenError("INVALID_QUANTITY", fmt.Sprintf("%d", item.Quantity))
		}

		key := itemKey{sku: item.SKU, epoch: item.Epoch}
		if _, ok := index[key]; !ok {
			order = append(order, key)
		}
		index[key] += item.Quantity
	}

	normalized := make([]TokenItem, 0, len(order))
	for _, key := range order {
		normalized = append(normalized, TokenItem{
			SKU:      key.sku,
			Epoch:    key.epoch,
			Quantity: index[key],
		})
	}
	return normalized, nil
}

func decodeLuaResult(result any) (string, []string, error) {
	switch v := result.(type) {
	case nil:
		return "", nil, errors.New("lua returned nil result")
	case string:
		return v, nil, nil
	case []byte:
		return string(v), nil, nil
	case []interface{}:
		if len(v) == 0 {
			return "", nil, errors.New("lua returned empty array")
		}
		status := fmt.Sprint(v[0])
		details := make([]string, 0, len(v)-1)
		for _, val := range v[1:] {
			details = append(details, fmt.Sprint(val))
		}
		return status, details, nil
	default:
		return fmt.Sprint(v), nil, nil
	}
}

func (m *defaultInventoryTokenModel) resolveEpochs(ctx context.Context, items []TokenItem) ([]TokenItem, error) {
	if len(items) == 0 {
		return nil, nil
	}

	resolved := make([]TokenItem, len(items))
	type skuIdxs struct {
		key   string
		index []int
	}

	pending := make(map[int64]*skuIdxs)
	order := make([]int64, 0)

	for i, item := range items {
		if item.SKU <= 0 {
			return nil, newTokenError("INVALID_SKU", fmt.Sprintf("%d", item.SKU))
		}
		if item.Quantity <= 0 {
			return nil, newTokenError("INVALID_QUANTITY", fmt.Sprintf("%d", item.Quantity))
		}
		resolved[i] = item
		if item.Epoch <= 0 {
			if pending[item.SKU] == nil {
				key := fmt.Sprintf(tokenEpochKeyPattern, item.SKU)
				pending[item.SKU] = &skuIdxs{
					key: key,
				}
				order = append(order, item.SKU)
			}
			pending[item.SKU].index = append(pending[item.SKU].index, i)
		}
	}

	if len(order) == 0 {
		return resolved, nil
	}

	keys := make([]string, 0, len(order))
	for _, sku := range order {
		keys = append(keys, pending[sku].key)
	}

	values, err := m.redis.MgetCtx(ctx, keys...)
	if err != nil {
		return nil, err
	}

	for idx, sku := range order {
		val := values[idx]
		if val == "" {
			return nil, newTokenError("NO_EPOCH", fmt.Sprintf("%d", sku))
		}
		epoch, convErr := strconv.ParseInt(val, 10, 64)
		if convErr != nil {
			return nil, fmt.Errorf("parse epoch for sku %d: %w", sku, convErr)
		}
		for _, pos := range pending[sku].index {
			resolved[pos].Epoch = epoch
		}
	}

	return resolved, nil
}

func decodeTicket(payload string) (*TokenTicket, error) {
	var ticket TokenTicket
	if err := json.Unmarshal([]byte(payload), &ticket); err != nil {
		return nil, fmt.Errorf("unmarshal ticket: %w", err)
	}
	return &ticket, nil
}

// AdmissionTicketKey returns the redis key used to store the admission ticket for a preorder.
func AdmissionTicketKey(preorderID string) string {
	return fmt.Sprintf(admissionTicketPattern, preorderID)
}

func (m *defaultInventoryTokenModel) ensureSnapshots(ctx context.Context, items []TokenItem) error {
	if len(items) == 0 || m.inventory == nil {
		return nil
	}

	seen := make(map[int64]struct{}, len(items))
	for _, item := range items {
		if item.SKU <= 0 {
			continue
		}
		if _, ok := seen[item.SKU]; ok {
			continue
		}
		seen[item.SKU] = struct{}{}
		if err := m.ensureSnapshotForSKU(ctx, item.SKU); err != nil {
			return err
		}
	}
	return nil
}

func (m *defaultInventoryTokenModel) ensureSnapshotForSKU(ctx context.Context, sku int64) error {
	if sku <= 0 {
		return newTokenError("INVALID_SKU", fmt.Sprintf("%d", sku))
	}

	epochKey := fmt.Sprintf(tokenEpochKeyPattern, sku)
	currentEpochStr, err := m.redis.GetCtx(ctx, epochKey)
	if err != nil {
		return err
	}

	if currentEpochStr == "" {
		return m.refreshSnapshotFromDB(ctx, sku, 0)
	}

	currentEpoch, err := strconv.ParseInt(currentEpochStr, 10, 64)
	if err != nil || currentEpoch <= 0 {
		return m.refreshSnapshotFromDB(ctx, sku, 0)
	}

	thresholdKey := fmt.Sprintf(tokenThresholdKeyPattern, sku, currentEpoch)
	exist, err := m.redis.ExistsCtx(ctx, thresholdKey)
	if err != nil {
		return err
	}

	if exist {
		return nil
	}

	return m.refreshSnapshotFromDB(ctx, sku, currentEpoch)
}

// 回源 DB 同步令牌数据
func (m *defaultInventoryTokenModel) refreshSnapshotFromDB(ctx context.Context, sku int64, oldEpoch int64) error {
	if m.inventory == nil {
		return newTokenError("SNAPSHOT_UNAVAILABLE")
	}

	record, err := m.inventory.FindOne(ctx, sku)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return newTokenError("SKU_NOT_FOUND", fmt.Sprintf("%d", sku))
		}
		return err
	}

	threshold := record.Stock
	return m.applySnapshot(ctx, sku, threshold, oldEpoch)
}

func (m *defaultInventoryTokenModel) applySnapshot(ctx context.Context, sku int64, threshold int64, oldEpoch int64) error {
	if sku <= 0 {
		return newTokenError("INVALID_SKU", fmt.Sprintf("%d", sku))
	}

	if threshold < 0 {
		threshold = 0
	}

	newEpoch := time.Now().UnixNano()
	if newEpoch <= 0 {
		newEpoch = time.Now().UnixNano()
	}

	epochStr := strconv.FormatInt(newEpoch, 10)
	thresholdStr := strconv.FormatInt(threshold, 10)
	thresholdKey := fmt.Sprintf(tokenThresholdKeyPattern, sku, newEpoch)
	issuedKey := fmt.Sprintf(tokenIssuedKeyPattern, sku, newEpoch)
	epochKey := fmt.Sprintf(tokenEpochKeyPattern, sku)

	err := m.redis.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		if err := pipe.Set(ctx, thresholdKey, thresholdStr, 0).Err(); err != nil {
			return err
		}
		if err := pipe.Set(ctx, issuedKey, "0", 0).Err(); err != nil {
			return err
		}
		if err := pipe.Set(ctx, epochKey, epochStr, 0).Err(); err != nil {
			return err
		}
		if oldEpoch > 0 && oldEpoch != newEpoch {
			oldThresholdKey := fmt.Sprintf(tokenThresholdKeyPattern, sku, oldEpoch)
			oldIssuedKey := fmt.Sprintf(tokenIssuedKeyPattern, sku, oldEpoch)
			if err := pipe.Del(ctx, oldThresholdKey, oldIssuedKey).Err(); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	logx.WithContext(ctx).Infow("inventory snapshot refreshed",
		logx.Field("sku", sku),
		logx.Field("epoch", newEpoch),
		logx.Field("threshold", threshold),
	)
	return nil
}
