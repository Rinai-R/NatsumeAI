// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package product

import (
	"context"

	"NatsumeAI/app/api/product/internal/logic/helper"
	"NatsumeAI/app/api/product/internal/svc"
	"NatsumeAI/app/api/product/internal/types"
	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/common/util"
	productsrv "NatsumeAI/app/services/product/productservice"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/x/errors"
)

type FeedProductsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFeedProductsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FeedProductsLogic {
	return &FeedProductsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FeedProductsLogic) FeedProducts(req *types.FeedProductsRequest) (resp *types.FeedProductsResponse, err error) {
	userID, err := util.UserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}

	if req != nil && req.Limit < 0 {
		return nil, errors.New(int(errno.InvalidParam), "invalid limit")
	}

	in := &productsrv.FeedProductsReq{
		UserId: userID,
	}

	if req != nil {
		in.Cursor = req.Cursor
		in.Limit = req.Limit
	}

	res, err := l.svcCtx.ProductRpc.FeedProducts(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: feed products rpc failed: ", err)
		return nil, err
	}

	if res == nil {
		return nil, errors.New(int(errno.InternalError), "empty feed products response")
	}

	if res.StatusCode != errno.StatusOK {
		l.Logger.Error("logic: feed products rpc returned error: ", res)
		return nil, errors.New(int(res.StatusCode), res.StatusMsg)
	}

	products := make([]types.ProductSummary, 0, len(res.Products))
	for _, item := range res.Products {
		products = append(products, helper.ToProductSummary(item))
	}

	resp = &types.FeedProductsResponse{
		Products:   products,
		NextCursor: res.NextCursor,
	}

	return
}
