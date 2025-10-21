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

type DeleteProductLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteProductLogic {
	return &DeleteProductLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteProductLogic) DeleteProduct(req *types.DeleteProductRequest) (resp *types.DeleteProductResponse,err error) {
	resp = &types.DeleteProductResponse{}
	if req == nil || req.ProductId <= 0 {
		return nil, errors.New(int(errno.InvalidParam), "invalid product id")
	}

	merchantID, err := util.UserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}

	in := &productsrv.DeleteProductReq{
		ProductId:  req.ProductId,
		MerchantId: merchantID,
	}

	res, err := l.svcCtx.ProductRpc.DeleteProduct(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: delete product rpc failed: ", err)
		return nil, err
	}

	if res == nil {
		return nil, errors.New(int(errno.InternalError), "empty delete product response")
	}

	resp.StatusCode = res.StatusCode
	resp.StatusMsg = res.StatusMsg

	return 
}
