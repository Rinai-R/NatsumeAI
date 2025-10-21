package logic

import (
	"context"

	"NatsumeAI/app/common/consts/errno"
	productmodel "NatsumeAI/app/dal/product"
	"NatsumeAI/app/services/product/internal/svc"
	"NatsumeAI/app/services/product/product"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddCategoriesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddCategoriesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddCategoriesLogic {
	return &AddCategoriesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 增加标签
func (l *AddCategoriesLogic) AddCategories(in *product.AddCategoriesReq) (*product.AddCategoriesResp, error) {
	resp := &product.AddCategoriesResp{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
	}

	if in == nil || in.GetProductId() <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid product id"
		return resp, nil
	}

	newCategories := sanitizeCategories(in.GetCategories())
	if len(newCategories) == 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "categories required"
		return resp, nil
	}

	record, err := l.svcCtx.ProductModel.FindOne(l.ctx, in.GetProductId())
	if err != nil {
		if err == productmodel.ErrNotFound {
			resp.StatusCode = errno.ProductNotFound
			resp.StatusMsg = "product not found"
			return resp, nil
		}
		l.Logger.Errorf("add categories find failed: %v", err)
		return resp, err
	}
	if record.MerchantId != in.MerchantId {
		l.Logger.Error("add tag merchantId mismatch")
		resp.StatusCode = errno.MerchantMismatch
		resp.StatusMsg = "merchant mismatch"
		return resp, nil
	}

	existing, err := categoriesFromNullString(record.Categories)
	if err != nil {
		l.Logger.Errorf("add categories decode existing failed: %v", err)
		return resp, err
	}

	merged := mergeCategories(existing, newCategories)

	categoriesValue, err := categoriesToNullString(merged)
	if err != nil {
		l.Logger.Errorf("add categories marshal failed: %v", err)
		return resp, err
	}

	record.Categories = categoriesValue

	if err := l.svcCtx.ProductModel.Update(l.ctx, record); err != nil {
		l.Logger.Errorf("add categories update failed: %v", err)
		return resp, err
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	return resp, nil
}
