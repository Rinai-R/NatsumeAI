package logic

import (
	"context"
	"strconv"

	"NatsumeAI/app/common/consts/errno"
	productmodel "NatsumeAI/app/dal/product"
	"NatsumeAI/app/services/inventory/inventory"
	"NatsumeAI/app/services/product/internal/svc"
	"NatsumeAI/app/services/product/product"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetProductLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetProductLogic {
	return &GetProductLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取商品详情
func (l *GetProductLogic) GetProduct(in *product.GetProductReq) (*product.GetProductResp, error) {
	resp := &product.GetProductResp{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
	}

	if in == nil || in.GetProductId() <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid product id"
		return resp, nil
	}

	if exist, _ := l.svcCtx.Bloom.
		ExistsCtx(l.ctx, []byte(strconv.Itoa(int(in.ProductId)))); !exist {

		resp.StatusCode = errno.ProductNotFound
		resp.StatusMsg = "product not found"
		return resp, nil
	}

	record, err := l.svcCtx.ProductModel.FindOne(l.ctx, in.GetProductId())
	if err != nil {
		if err == productmodel.ErrNotFound {
			resp.StatusCode = errno.ProductNotFound
			resp.StatusMsg = "product not found"
			return resp, nil
		}
		l.Logger.Errorf("get product failed: %v", err)
		return resp, err
	}

	protoProduct, err := productModelToProto(record)
	if err != nil {
		l.Logger.Errorf("get product convert proto failed: %v", err)
		return resp, err
	}
	if categoryRecords, err := l.svcCtx.ProductCategoriesModel.ListByProductId(l.ctx, in.GetProductId()); err != nil {
		if err != productmodel.ErrNotFound {
			l.Logger.Errorf("get product list categories failed: %v", err)
			return resp, err
		}
	} else {
		protoProduct.Categories = categoriesFromRecords(categoryRecords)
	}

	if resp, err := l.svcCtx.InventoryRpc.GetInventory(l.ctx, &inventory.GetInventoryReq{
		ProductIds: []int64{in.ProductId},
	}); err == nil && resp.StatusCode == errno.StatusOK && len(resp.Items) > 0 {
		protoProduct.Stock = resp.Items[0].Inventory
		protoProduct.Sold = resp.Items[0].SoldCount
	} else {
		// 获取库存失败仅作降级打日志，不影响获取其他信息
		l.Logger.Error("获取库存失败，rpc 返回信息：", resp, err)
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	resp.Product = protoProduct

	return resp, nil
}
