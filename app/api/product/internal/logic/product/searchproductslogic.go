// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package product

import (
	"context"
	"strings"

	"NatsumeAI/app/api/product/internal/logic/helper"
	"NatsumeAI/app/api/product/internal/svc"
	"NatsumeAI/app/api/product/internal/types"
	"NatsumeAI/app/common/consts/errno"
	productsrv "NatsumeAI/app/services/product/productservice"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/x/errors"
)

type SearchProductsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchProductsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchProductsLogic {
	return &SearchProductsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchProductsLogic) SearchProducts(req *types.SearchProductsRequest) (resp *types.SearchProductsResponse, err error) {
	if req != nil {
		if req.MinPrice < 0 || req.MaxPrice < 0 {
			return nil, errors.New(int(errno.InvalidParam), "invalid price range")
		}
		if req.MinPrice > 0 && req.MaxPrice > 0 && req.MinPrice > req.MaxPrice {
			return nil, errors.New(int(errno.InvalidParam), "min price greater than max price")
		}
		if req.Limit < 0 || req.Offset < 0 {
			return nil, errors.New(int(errno.InvalidParam), "invalid pagination parameters")
		}
	}

	in := &productsrv.SearchProductsReq{}
	if req != nil {
		in.Keyword = strings.TrimSpace(req.Keyword)
		in.MinPrice = req.MinPrice
		in.MaxPrice = req.MaxPrice
		in.Limit = req.Limit
		in.Offset = req.Offset
		in.SortBy = strings.TrimSpace(req.SortBy)
		in.SortOrder = strings.TrimSpace(req.SortOrder)

		if len(req.Categories) > 0 {
			in.Categories = make([]string, 0, len(req.Categories))
			for _, c := range req.Categories {
				if trimmed := strings.TrimSpace(c); trimmed != "" {
					in.Categories = append(in.Categories, trimmed)
				}
			}
		}
	}

	res, err := l.svcCtx.ProductRpc.SearchProducts(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: search products rpc failed: ", err)
		return nil, err
	}

	if res == nil {
		return nil, errors.New(int(errno.InternalError), "empty search products response")
	}

	if res.StatusCode != errno.StatusOK {
		l.Logger.Error("logic: search products rpc returned error: ", res)
		return nil, errors.New(int(res.StatusCode), res.StatusMsg)
	}

	summaries := make([]types.ProductSummary, 0, len(res.Products))
	for _, item := range res.Products {
		summaries = append(summaries, helper.ToProductSummary(item))
	}

	resp = &types.SearchProductsResponse{
		Products:   summaries,
		TotalCount: res.TotalCount,
	}

	return
}
