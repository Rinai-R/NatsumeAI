package es

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
)

// ErrIncompatibleEmbeddingMapping indicates the existing index mapping is not suitable for vector search.
var ErrIncompatibleEmbeddingMapping = errors.New("embedding mapping incompatible with vector search requirements")

// ProductIndexParams describes the configuration required to ensure the product index exists.
type ProductIndexParams struct {
	IndexName        string
	EmbeddingDims    int
	NumberOfShards   int
	NumberOfReplicas int
}

// ProductIndexInfo reports the detected capabilities of the ensured index.
type ProductIndexInfo struct {
	SupportsVector bool
	EmbeddingDims  int
}

// EnsureProductIndex creates the product index when missing and validates the embedding mapping.
func EnsureProductIndex(ctx context.Context, client *elasticsearch.Client, params ProductIndexParams) (ProductIndexInfo, error) {
	info := ProductIndexInfo{EmbeddingDims: params.EmbeddingDims}

	if client == nil || strings.TrimSpace(params.IndexName) == "" {
		return info, fmt.Errorf("missing elasticsearch client or index name")
	}

	if params.EmbeddingDims <= 0 {
		return info, fmt.Errorf("invalid embedding dimension %d", params.EmbeddingDims)
	}

	shards := params.NumberOfShards
	if shards <= 0 {
		shards = 1
	}
	replicas := params.NumberOfReplicas
	if replicas < 0 {
		replicas = 0
	}

	existsRes, err := client.Indices.Exists(
		[]string{params.IndexName},
		client.Indices.Exists.WithContext(ctx),
	)
	if err != nil {
		return info, fmt.Errorf("check index existence: %w", err)
	}
	defer existsRes.Body.Close()

	if existsRes.StatusCode == http.StatusNotFound {
		body, err := buildProductIndexDefinition(params.EmbeddingDims, shards, replicas)
		if err != nil {
			return info, err
		}

		createRes, err := client.Indices.Create(
			params.IndexName,
			client.Indices.Create.WithContext(ctx),
			client.Indices.Create.WithBody(bytes.NewReader(body)),
		)
		if err != nil {
			return info, fmt.Errorf("create index: %w", err)
		}
		defer createRes.Body.Close()

		if createRes.IsError() {
			resp, _ := io.ReadAll(createRes.Body)
			return info, fmt.Errorf("create index status %s: %s", createRes.Status(), strings.TrimSpace(string(resp)))
		}

		info.SupportsVector = true
		return info, nil
	}

	if existsRes.IsError() {
		resp, _ := io.ReadAll(existsRes.Body)
		return info, fmt.Errorf("index existence status %s: %s", existsRes.Status(), strings.TrimSpace(string(resp)))
	}

	info.SupportsVector = mappingSupportsVector(ctx, client, params.IndexName, params.EmbeddingDims)
	if !info.SupportsVector {
		return info, ErrIncompatibleEmbeddingMapping
	}

	return info, nil
}

func buildProductIndexDefinition(dims, shards, replicas int) ([]byte, error) {
	definition := map[string]any{
		"settings": map[string]any{
			"number_of_shards":   shards,
			"number_of_replicas": replicas,
		},
		"mappings": map[string]any{
			"properties": map[string]any{
				"product_id": map[string]any{
					"type": "long",
				},
				"merchant_id": map[string]any{
					"type": "long",
				},
				"name": map[string]any{
					"type": "text",
					"fields": map[string]any{
						"keyword": map[string]any{
							"type":         "keyword",
							"ignore_above": 256,
						},
					},
				},
				"description": map[string]any{
					"type": "text",
				},
				"picture": map[string]any{
					"type": "keyword",
				},
				"price": map[string]any{
					"type": "double",
				},
				"categories": map[string]any{
					"type": "keyword",
				},
				"category_updated_at": map[string]any{
					"type":   "date",
					"format": "strict_date_optional_time||epoch_millis",
				},
				"created_at": map[string]any{
					"type":   "date",
					"format": "strict_date_optional_time||epoch_millis",
				},
				"updated_at": map[string]any{
					"type":   "date",
					"format": "strict_date_optional_time||epoch_millis",
				},
				"embedding": map[string]any{
					"type":       "dense_vector",
					"dims":       dims,
					"index":      true,
					"similarity": "cosine",
					"index_options": map[string]any{
						"type":            "hnsw",
						"m":               16,
						"ef_construction": 100,
					},
				},
			},
		},
	}

	payload, err := json.Marshal(definition)
	if err != nil {
		return nil, fmt.Errorf("encode index definition: %w", err)
	}
	return payload, nil
}

func mappingSupportsVector(ctx context.Context, client *elasticsearch.Client, indexName string, expectedDims int) bool {
	res, err := client.Indices.GetMapping(
		client.Indices.GetMapping.WithContext(ctx),
		client.Indices.GetMapping.WithIndex(indexName),
	)
	if err != nil {
		return false
	}
	defer res.Body.Close()

	if res.IsError() {
		return false
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return false
	}

	var mapping map[string]struct {
		Mappings struct {
			Properties map[string]struct {
				Type       string `json:"type"`
				Dims       int    `json:"dims"`
				Similarity string `json:"similarity"`
			} `json:"properties"`
		} `json:"mappings"`
	}
	if err := json.Unmarshal(data, &mapping); err != nil {
		return false
	}

	indexMapping, ok := mapping[indexName]
	if !ok {
		for _, v := range mapping {
			indexMapping = v
			ok = true
			break
		}
	}
	if !ok {
		return false
	}

	embedding, ok := indexMapping.Mappings.Properties["embedding"]
	if !ok {
		return false
	}
	if embedding.Type != "dense_vector" {
		return false
	}
	if expectedDims > 0 && embedding.Dims != expectedDims {
		return false
	}

	return true
}
