// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package product_manage

import (
	"context"
	"strings"

	"NatsumeAI/app/api/product/internal/logic/helper"
	"NatsumeAI/app/api/product/internal/svc"
	"NatsumeAI/app/api/product/internal/types"
	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/common/util"
	productsrv "NatsumeAI/app/services/product/productservice"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/x/errors"
)

type UpdateProductLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateProductLogic {
	return &UpdateProductLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateProductLogic) UpdateProduct(req *types.UpdateProductRequest) (resp *types.UpdateProductResponse, err error) {
	resp = &types.UpdateProductResponse{}
	if req == nil || req.ProductId <= 0 {
		return nil, errors.New(int(errno.InvalidParam), "invalid product id")
	}

	merchantID, err := util.UserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	description := strings.TrimSpace(req.Description)
	picture := strings.TrimSpace(req.Picture)

	if name == "" || description == "" || picture == "" || req.Price <= 0 {
		return nil, errors.New(int(errno.InvalidParam), "invalid product payload")
	}

	in := &productsrv.UpdateProductReq{
		ProductId:   req.ProductId,
		Name:        name,
		Description: description,
		Picture:     picture,
		Price:       req.Price,
		MerchantId:  merchantID,
	}

	res, err := l.svcCtx.ProductRpc.UpdateProduct(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: update product rpc failed: ", err)
		return nil, err
	}

	if res == nil {
		return nil, errors.New(int(errno.InternalError), "empty update product response")
	}

	if res.StatusCode != errno.StatusOK {
		l.Logger.Error("logic: update product rpc returned error: ", res)
		return nil, errors.New(int(res.StatusCode), res.StatusMsg)
	}

	product := helper.ToProduct(res.Product)
	resp.StatusCode = res.StatusCode
	resp.StatusMsg = res.StatusMsg
	resp.Product = product

	return
}
