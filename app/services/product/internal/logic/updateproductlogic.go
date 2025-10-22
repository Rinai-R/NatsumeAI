package logic

import (
	"context"
	"strings"

	"NatsumeAI/app/common/consts/errno"
	productmodel "NatsumeAI/app/dal/product"
	"NatsumeAI/app/services/product/internal/svc"
	"NatsumeAI/app/services/product/product"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateProductLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateProductLogic {
	return &UpdateProductLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 更新商品信息，商家调用
func (l *UpdateProductLogic) UpdateProduct(in *product.UpdateProductReq) (*product.UpdateProductResp, error) {
	resp := &product.UpdateProductResp{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
	}

	if in == nil || in.GetProductId() <= 0 || in.GetMerchantId() <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid request payload"
		return resp, nil
	}

	name := strings.TrimSpace(in.GetName())
	description := strings.TrimSpace(in.GetDescription())
	picture := strings.TrimSpace(in.GetPicture())
	price := in.GetPrice()
	merchantID := in.GetMerchantId()

	if name == "" || description == "" || picture == "" || price <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid product payload"
		return resp, nil
	}

	record, err := l.svcCtx.ProductModel.FindOne(l.ctx, in.GetProductId())
	if err != nil {
		if err == productmodel.ErrNotFound {
			resp.StatusCode = errno.ProductNotFound
			resp.StatusMsg = "product not found"
			return resp, nil
		}
		l.Logger.Errorf("update product find failed: %v", err)
		return resp, err
	}

	if record.MerchantId != merchantID {
		resp.StatusCode = errno.MerchantMismatch
		resp.StatusMsg = "merchant mismatch"
		return resp, nil
	}

	record.Name = name
	record.Description = description
	record.Picture = picture
	record.Price = price

	if err := l.svcCtx.ProductModel.Update(l.ctx, record); err != nil {
		l.Logger.Errorf("update product failed: %v", err)
		return resp, err
	}

	updated, err := l.svcCtx.ProductModel.FindOne(l.ctx, in.GetProductId())
	if err != nil {
		l.Logger.Errorf("update product refetch failed: %v", err)
		return resp, err
	}

	protoProduct, err := productModelToProto(updated)
	if err != nil {
		l.Logger.Errorf("update product convert proto failed: %v", err)
		return resp, err
	}
	if categoryRecords, err := l.svcCtx.ProductCategoriesModel.ListByProductId(l.ctx, in.GetProductId()); err != nil {
		if err != productmodel.ErrNotFound {
			l.Logger.Errorf("update product list categories failed: %v", err)
			return resp, err
		}
	} else {
		protoProduct.Categories = categoriesFromRecords(categoryRecords)
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	resp.Product = protoProduct

	return resp, nil
}
