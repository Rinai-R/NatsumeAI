// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package product_manage

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

type CreateProductLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateProductLogic {
	return &CreateProductLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateProductLogic) CreateProduct(req *types.CreateProductRequest) (resp *types.CreateProductResponse, err error) {
	resp = &types.CreateProductResponse{}
	if req == nil {
		return nil, errors.New(int(errno.InvalidParam), "missing request payload")
	}

	merchantID, err := util.UserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}


	if req.Name == "" || req.Description == "" || req.Picture == "" || req.Price <= 0 || req.Stock < 0 {
		return nil, errors.New(int(errno.InvalidParam), "invalid product payload")
	}

	in := &productsrv.CreateProductReq{
		Name:        req.Name,
		Description: req.Description,
		Picture:     req.Picture,
		Price:       req.Price,
		Stock:       req.Stock,
		Categories:  req.Categories,
		MerchantId:  merchantID,
	}

	res, err := l.svcCtx.ProductRpc.CreateProduct(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: create product rpc failed: ", err)
		return nil, err
	}

	if res == nil {
		return nil, errors.New(int(errno.InternalError), "empty create product response")
	}

	product := helper.ToProduct(res.Product)
	resp.Product = product
	resp.StatusCode = res.StatusCode
	resp.StatusMsg = res.StatusMsg

	return
}
