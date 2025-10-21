package logic

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	productmodel "NatsumeAI/app/dal/product"
	"NatsumeAI/app/services/product/product"
)

const productTimeFormat = time.RFC3339

func sanitizeCategories(categories []string) []string {
	if len(categories) == 0 {
		return nil
	}

	dedup := make(map[string]struct{}, len(categories))
	normalized := make([]string, 0, len(categories))
	for _, category := range categories {
		c := strings.TrimSpace(category)
		if c == "" {
			continue
		}
		if _, ok := dedup[c]; ok {
			continue
		}
		dedup[c] = struct{}{}
		normalized = append(normalized, c)
	}

	if len(normalized) == 0 {
		return nil
	}

	return normalized
}

func categoriesToNullString(categories []string) (sql.NullString, error) {
	normalized := sanitizeCategories(categories)
	if len(normalized) == 0 {
		return sql.NullString{}, nil
	}

	raw, err := json.Marshal(normalized)
	if err != nil {
		return sql.NullString{}, err
	}

	return sql.NullString{String: string(raw), Valid: true}, nil
}

func categoriesFromNullString(value sql.NullString) ([]string, error) {
	if !value.Valid {
		return nil, nil
	}

	raw := strings.TrimSpace(value.String)
	if raw == "" {
		return nil, nil
	}

	var categories []string
	if err := json.Unmarshal([]byte(raw), &categories); err != nil {
		return nil, err
	}

	return sanitizeCategories(categories), nil
}

func productModelToProto(model *productmodel.Products) (*product.Product, error) {
	if model == nil {
		return nil, nil
	}

	categories, err := categoriesFromNullString(model.Categories)
	if err != nil {
		return nil, err
	}

	createdAt := model.CreatedAt.UTC().Format(productTimeFormat)
	updatedAt := model.UpdatedAt.UTC().Format(productTimeFormat)

	return &product.Product{
		Id:          model.Id,
		MerchantId:  model.MerchantId,
		Name:        model.Name,
		Description: model.Description,
		Picture:     model.Picture,
		Price:       model.Price,
		Categories:  categories,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func mergeCategories(existing []string, additions []string) []string {
	base := sanitizeCategories(existing)
	extra := sanitizeCategories(additions)

	if len(base) == 0 {
		base = nil
	}

	if len(extra) == 0 {
		return base
	}

	set := make(map[string]struct{}, len(base)+len(extra))
	merged := make([]string, 0, len(base)+len(extra))

	for _, c := range base {
		if _, ok := set[c]; ok {
			continue
		}
		set[c] = struct{}{}
		merged = append(merged, c)
	}

	for _, c := range extra {
		if _, ok := set[c]; ok {
			continue
		}
		set[c] = struct{}{}
		merged = append(merged, c)
	}

	if len(merged) == 0 {
		return nil
	}

	return merged
}
