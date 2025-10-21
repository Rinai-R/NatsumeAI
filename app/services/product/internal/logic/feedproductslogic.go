package logic

import (
	"context"

	"NatsumeAI/app/services/product/internal/svc"
	"NatsumeAI/app/services/product/product"

	"github.com/zeromicro/go-zero/core/logx"
)

type FeedProductsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFeedProductsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FeedProductsLogic {
	return &FeedProductsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 首页推荐商品
func (l *FeedProductsLogic) FeedProducts(in *product.FeedProductsReq) (*product.FeedProductsResp, error) {
	// todo: add your logic here and delete this line

	return &product.FeedProductsResp{}, nil
}
