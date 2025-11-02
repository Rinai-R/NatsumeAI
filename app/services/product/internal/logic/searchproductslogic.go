package logic

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/services/product/internal/svc"
	"NatsumeAI/app/services/product/product"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchProductsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSearchProductsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchProductsLogic {
	return &SearchProductsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 搜索商品
func (l *SearchProductsLogic) SearchProducts(in *product.SearchProductsReq) (*product.SearchProductsResp, error) {
	resp := &product.SearchProductsResp{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
	}

	if in == nil {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid request"
		return resp, nil
	}

	if l.svcCtx.ESClient == nil {
		resp.StatusMsg = "search service unavailable"
		return resp, nil
	}

	pageSize := in.GetLimit()
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	offset := in.GetOffset()
	if offset < 0 {
		offset = 0
	}

	candidateSize := int64(pageSize + offset)
	if candidateSize < 200 {
		candidateSize = 200
	}

	keywordMust, keywordFilter := buildQueryClauses(in)
	textResponse, err := l.searchKeywordCandidates(keywordMust, keywordFilter, candidateSize)
	if err != nil {
		return resp, err
	}

	vectorScores := make(map[string]float64)
	vectorDocs := make(map[string]esHitSource)
	vector, embedErr := l.computeQueryEmbedding(in)
	if embedErr != nil {
		l.Logger.Errorf("generate query embedding failed, fallback to deterministic vector: %v", embedErr)
	}
	if len(vector) > 0 {
		if err := l.searchVectorCandidates(vector, keywordFilter, candidateSize, vectorScores, vectorDocs); err != nil {
			return resp, err
		}
	}

	textScores := make(map[string]float64)
	documents := make(map[string]esHitSource)

	for _, hit := range textResponse.Hits.Hits {
		textScores[hit.ID] = hit.Score
		documents[hit.ID] = hit.Source
	}
	for id, doc := range vectorDocs {
		if _, ok := documents[id]; !ok {
			documents[id] = doc
		}
	}

	if len(documents) == 0 {
		resp.StatusCode = errno.StatusOK
		resp.StatusMsg = "ok"
		resp.Products = []*product.ProductSummary{}
		return resp, nil
	}

	normText := normalizeScores(textScores)
	normVector := normalizeScores(vectorScores)

	alpha := l.svcCtx.HybridAlpha()
	type candidate struct {
		id    string
		score float64
	}
	candidates := make([]candidate, 0, len(documents))
	for id := range documents {
		score := alpha*normText[id] + (1-alpha)*normVector[id]
		candidates = append(candidates, candidate{id: id, score: score})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].id < candidates[j].id
		}
		return candidates[i].score > candidates[j].score
	})

	start := int(offset)
	if start > len(candidates) {
		start = len(candidates)
	}
	end := start + int(pageSize)
	if end > len(candidates) {
		end = len(candidates)
	}

	resp.Products = make([]*product.ProductSummary, 0, end-start)
	for _, cand := range candidates[start:end] {
		if src, ok := documents[cand.id]; ok {
			resp.Products = append(resp.Products, &product.ProductSummary{
				Id:          src.ProductID,
				Name:        src.Name,
				Description: src.Description,
				Picture:     src.Picture,
				Price:       src.Price,
				Categories:  append([]string{}, src.Categories...),
				MerchantId:  src.MerchantID,
			})
		}
	}

	resp.TotalCount = textResponse.Hits.Total.Value
	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"

	return resp, nil
}

func buildQueryClauses(in *product.SearchProductsReq) (must []any, filter []any) {
	if keyword := strings.TrimSpace(in.GetKeyword()); keyword != "" {
		must = append(must, map[string]any{
			"multi_match": map[string]any{
				"query":  keyword,
				"fields": []string{"name^2", "description", "categories"},
			},
		})
	}

	if categories := in.GetCategories(); len(categories) > 0 {
		filter = append(filter, map[string]any{
			"terms": map[string]any{
				"categories": categories,
			},
		})
	}

	rangeConditions := map[string]any{}
	if minPrice := in.GetMinPrice(); minPrice > 0 {
		rangeConditions["gte"] = minPrice
	}
	if maxPrice := in.GetMaxPrice(); maxPrice > 0 {
		rangeConditions["lte"] = maxPrice
	}
	if len(rangeConditions) > 0 {
		filter = append(filter, map[string]any{
			"range": map[string]any{
				"price": rangeConditions,
			},
		})
	}

	return must, filter
}

func buildSearchQuery(in *product.SearchProductsReq) map[string]any {
	must, filter := buildQueryClauses(in)
	boolQuery := map[string]any{}

	if len(must) > 0 {
		boolQuery["must"] = must
	}
	if len(filter) > 0 {
		boolQuery["filter"] = filter
	}
	if len(boolQuery) == 0 {
		boolQuery["must"] = []any{map[string]any{
			"match_all": map[string]any{},
		}}
	}

	return map[string]any{
		"bool": boolQuery,
	}
}

type esSearchResponse struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			ID     string      `json:"_id"`
			Score  float64     `json:"_score"`
			Source esHitSource `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

type esHitSource struct {
	ProductID   int64    `json:"product_id"`
	MerchantID  int64    `json:"merchant_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Picture     string   `json:"picture"`
	Price       int64    `json:"price"`
	Categories  []string `json:"categories"`
}

func (l *SearchProductsLogic) computeQueryEmbedding(in *product.SearchProductsReq) ([]float64, error) {
	dim := l.svcCtx.EmbeddingDimension()
	if dim <= 0 {
		return nil, nil
	}

	textParts := []string{in.GetKeyword()}
	if len(in.GetCategories()) > 0 {
		textParts = append(textParts, strings.Join(in.GetCategories(), " "))
	}
	text := strings.TrimSpace(strings.Join(textParts, " "))
	if text == "" {
		return nil, nil
	}

	fallback := deterministicEmbedding(text, dim)
	if l.svcCtx.Embedder == nil {
		return fallback, nil
	}

	embeddings, err := l.svcCtx.Embedder.EmbedStrings(l.ctx, []string{text})
	if err != nil {
		return fallback, err
	}
	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return fallback, fmt.Errorf("empty embedding response")
	}
	if expected := l.svcCtx.EmbeddingDimension(); expected > 0 && len(embeddings[0]) != expected {
		return fallback, fmt.Errorf("embedding dimension mismatch, expect %d got %d", expected, len(embeddings[0]))
	}
	return embeddings[0], nil
}

func deterministicEmbedding(text string, dim int) []float64 {
	if dim <= 0 {
		return nil
	}
	if text == "" {
		text = "empty"
	}
	vec := make([]float64, dim)
	for i := 0; i < dim; i++ {
		hasher := sha256.Sum256(append([]byte(text), byte(i)))
		val := binary.LittleEndian.Uint32(hasher[:4])
		vec[i] = (float64(int32(val%20000)-10000) / 10000.0)
	}
	normalizeVector64(vec)
	return vec
}

func normalizeVector64(vec []float64) {
	var sum float64
	for _, v := range vec {
		sum += v * v
	}
	if sum <= 0 {
		return
	}
	norm := math.Sqrt(sum)
	if norm == 0 {
		return
	}
	for i := range vec {
		vec[i] /= norm
	}
}

func normalizeScores(scores map[string]float64) map[string]float64 {
	if len(scores) == 0 {
		return map[string]float64{}
	}
	var minScore, maxScore float64
	first := true
	for _, score := range scores {
		if first {
			minScore, maxScore = score, score
			first = false
			continue
		}
		if score < minScore {
			minScore = score
		}
		if score > maxScore {
			maxScore = score
		}
	}
	normalized := make(map[string]float64, len(scores))
	if maxScore-minScore <= 1e-9 {
		for id := range scores {
			normalized[id] = 1
		}
		return normalized
	}
	for id, score := range scores {
		normalized[id] = (score - minScore) / (maxScore - minScore)
	}
	return normalized
}

func (l *SearchProductsLogic) searchKeywordCandidates(must, filter []any, size int64) (esSearchResponse, error) {
	query := map[string]any{
		"bool": map[string]any{},
	}
	if len(must) > 0 {
		query["bool"].(map[string]any)["must"] = must
	}
	if len(filter) > 0 {
		query["bool"].(map[string]any)["filter"] = filter
	}

	body := map[string]any{
		"query": query,
		"size":  size,
	}

	if payload, err := json.Marshal(body); err != nil {
		return esSearchResponse{}, fmt.Errorf("encode keyword search body failed: %w", err)
	} else {
		res, err := l.svcCtx.ESClient.Search(
			l.svcCtx.ESClient.Search.WithContext(l.ctx),
			l.svcCtx.ESClient.Search.WithIndex(l.svcCtx.ProductIndexName()),
			l.svcCtx.ESClient.Search.WithBody(bytes.NewReader(payload)),
		)
		if err != nil {
			return esSearchResponse{}, fmt.Errorf("es keyword search failed: %w", err)
		}
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return esSearchResponse{}, fmt.Errorf("read keyword search response failed: %w", err)
		}
		if res.IsError() {
			return esSearchResponse{}, fmt.Errorf("es keyword search status %s: %s", res.Status(), strings.TrimSpace(string(data)))
		}
		var parsed esSearchResponse
		if err := json.Unmarshal(data, &parsed); err != nil {
			return esSearchResponse{}, fmt.Errorf("decode keyword response failed: %w", err)
		}
		return parsed, nil
	}
}

func (l *SearchProductsLogic) searchVectorCandidates(vector []float64, filter []any, size int64, scores map[string]float64, docs map[string]esHitSource) error {
	if len(vector) == 0 {
		return nil
	}

	knn := map[string]any{
		"field":          "embedding",
		"query_vector":   vector,
		"k":              size,
		"num_candidates": size,
	}

	if len(filter) > 0 {
		knn["filter"] = map[string]any{
			"bool": map[string]any{
				"filter": filter,
			},
		}
	}

	body := map[string]any{
		"knn":  knn,
		"size": size,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode vector search body failed: %w", err)
	}

	res, err := l.svcCtx.ESClient.Search(
		l.svcCtx.ESClient.Search.WithContext(l.ctx),
		l.svcCtx.ESClient.Search.WithIndex(l.svcCtx.ProductIndexName()),
		l.svcCtx.ESClient.Search.WithBody(bytes.NewReader(payload)),
	)
	if err != nil {
		return fmt.Errorf("es vector search failed: %w", err)
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read vector search response failed: %w", err)
	}
	if res.IsError() {
		return fmt.Errorf("es vector search status %s: %s", res.Status(), strings.TrimSpace(string(data)))
	}

	var parsed esSearchResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("decode vector response failed: %w", err)
	}

	for _, hit := range parsed.Hits.Hits {
		scores[hit.ID] = hit.Score
		docs[hit.ID] = hit.Source
	}

	return nil
}

func float32SliceTo64(src []float32) []float64 {
	if len(src) == 0 {
		return nil
	}
	dst := make([]float64, len(src))
	for i, v := range src {
		dst[i] = float64(v)
	}
	return dst
}
