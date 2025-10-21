package helper

import (
	"NatsumeAI/app/api/product/internal/types"
	productsrv "NatsumeAI/app/services/product/productservice"
)

func ToProduct(src *productsrv.Product) types.Product {
	if src == nil {
		return types.Product{}
	}

	dst := types.Product{
		ProductId:   src.Id,
		MerchantId:  src.MerchantId,
		Name:        src.Name,
		Description: src.Description,
		Picture:     src.Picture,
		Price:       src.Price,
		Stock:       src.Stock,
		Sold:        src.Sold,
		CreatedAt:   src.CreatedAt,
		UpdatedAt:   src.UpdatedAt,
	}

	if len(src.Categories) > 0 {
		dst.Categories = append([]string(nil), src.Categories...)
	}

	return dst
}

func ToProductSummary(src *productsrv.ProductSummary) types.ProductSummary {
	if src == nil {
		return types.ProductSummary{}
	}

	dst := types.ProductSummary{
		ProductId:   src.Id,
		MerchantId:  src.MerchantId,
		Name:        src.Name,
		Description: src.Description,
		Picture:     src.Picture,
		Price:       src.Price,
	}

	if len(src.Categories) > 0 {
		dst.Categories = append([]string(nil), src.Categories...)
	}

	return dst
}
