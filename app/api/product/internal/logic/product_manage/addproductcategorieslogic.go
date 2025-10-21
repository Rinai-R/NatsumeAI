// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package product_manage

import (
	"context"

	"NatsumeAI/app/api/product/internal/svc"
	"NatsumeAI/app/api/product/internal/types"
	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/common/util"
	productsrv "NatsumeAI/app/services/product/productservice"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/x/errors"
)

type AddProductCategoriesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddProductCategoriesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddProductCategoriesLogic {
	return &AddProductCategoriesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 只增标签说是
func (l *AddProductCategoriesLogic) AddProductCategories(req *types.AddCategoriesRequest) (resp *types.AddCategoriesResponse, err error) {
	resp = &types.AddCategoriesResponse{}
	if req == nil || req.ProductId <= 0 {
		return nil, errors.New(int(errno.InvalidParam), "invalid product id")
	}

	if len(req.Categories) == 0 {
		return nil, errors.New(int(errno.InvalidParam), "categories required")
	}

	merchantId, err := util.UserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}


	in := &productsrv.AddCategoriesReq{
		ProductId:  req.ProductId,
		MerchantId: merchantId,
		Categories: req.Categories,
	}

	res, err := l.svcCtx.ProductRpc.AddCategories(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: add product categories rpc failed: ", err)
		return nil, err
	}

	if res == nil {
		return nil, errors.New(int(errno.InternalError), "empty add categories response")
	}

	resp.StatusCode = res.StatusCode
	resp.StatusMsg = res.StatusMsg

	return
}
