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

type GetProductLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetProductLogic {
	return &GetProductLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetProductLogic) GetProduct(req *types.GetProductRequest) (resp *types.GetProductResponse, err error) {
	if req == nil || req.ProductId <= 0 {
		return nil, errors.New(int(errno.InvalidParam), "invalid product id")
	}

	userID, err := util.UserIdFromCtx(l.ctx)
	if err != nil {
		if req.UserId > 0 {
			userID = req.UserId
		} else {
			userID = 0
		}
	}

	in := &productsrv.GetProductReq{
		ProductId: req.ProductId,
		UserId:    userID,
	}

	res, err := l.svcCtx.ProductRpc.GetProduct(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: get product rpc failed: ", err)
		return nil, err
	}

	if res == nil {
		return nil, errors.New(int(errno.InternalError), "empty get product response")
	}

	if res.StatusCode != errno.StatusOK {
		l.Logger.Error("logic: get product rpc returned error: ", res)
		return nil, errors.New(int(res.StatusCode), res.StatusMsg)
	}

	product := helper.ToProduct(res.Product)
	resp = &types.GetProductResponse{
		Product: product,
	}

	return
}
