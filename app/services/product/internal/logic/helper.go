package logic

import (
	"strings"
	"time"

	productmodel "NatsumeAI/app/dal/product"
	"NatsumeAI/app/services/product/product"
)

const (
	productTimeFormat = time.RFC3339
	maxCategoryLength = 64
)

func productModelToProto(model *productmodel.Products) (*product.Product, error) {
	if model == nil {
		return nil, nil
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
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func sanitizeCategories(categories []string) []string {
	if len(categories) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(categories))
	result := make([]string, 0, len(categories))
	for _, raw := range categories {
		c := strings.TrimSpace(raw)
		if c == "" {
			continue
		}
		if len(c) > maxCategoryLength {
			runes := []rune(c)
			if len(runes) > maxCategoryLength {
				c = string(runes[:maxCategoryLength])
			} else {
				c = string(runes)
			}
		}
		if _, exists := seen[c]; exists {
			continue
		}
		seen[c] = struct{}{}
		result = append(result, c)
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func categoriesFromRecords(records []*productmodel.ProductCategories) []string {
	if len(records) == 0 {
		return nil
	}

	result := make([]string, 0, len(records))
	for _, rec := range records {
		if rec == nil || rec.Category == "" {
			continue
		}
		result = append(result, rec.Category)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
