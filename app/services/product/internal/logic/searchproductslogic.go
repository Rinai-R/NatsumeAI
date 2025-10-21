package logic

import (
	"context"

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
	// todo: add your logic here and delete this line

	return &product.SearchProductsResp{}, nil
}
