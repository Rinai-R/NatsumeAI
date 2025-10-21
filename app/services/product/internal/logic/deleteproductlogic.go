package logic

import (
	"context"

	"NatsumeAI/app/common/consts/errno"
	productmodel "NatsumeAI/app/dal/product"
	inventorypb "NatsumeAI/app/services/inventory/inventory"
	"NatsumeAI/app/services/product/internal/svc"
	"NatsumeAI/app/services/product/product"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteProductLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteProductLogic {
	return &DeleteProductLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 删除商品，商家调用
func (l *DeleteProductLogic) DeleteProduct(in *product.DeleteProductReq) (*product.DeleteProductResp, error) {
	resp := &product.DeleteProductResp{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
	}

	if in == nil || in.GetProductId() <= 0 || in.GetMerchantId() <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid request payload"
		return resp, nil
	}

	record, err := l.svcCtx.ProductModel.FindOne(l.ctx, in.GetProductId())
	if err != nil {
		if err == productmodel.ErrNotFound {
			resp.StatusCode = errno.ProductNotFound
			resp.StatusMsg = "product not found"
			return resp, nil
		}
		l.Logger.Errorf("delete product find failed: %v", err)
		return resp, err
	}

	if record.MerchantId != in.GetMerchantId() {
		resp.StatusCode = errno.MerchantMismatch
		resp.StatusMsg = "merchant mismatch"
		return resp, nil
	}

	inventoryResp, err := l.svcCtx.InventoryRpc.DeleteInventory(l.ctx, &inventorypb.DeleteInventoryReq{
		ProductId:  in.GetProductId(),
		MerchantId: in.GetMerchantId(),
	})
	if err != nil {
		l.Logger.Errorf("delete product inventory rpc failed: %v", err)
		resp.StatusCode = errno.InternalError
		resp.StatusMsg = "delete inventory failed"
		return resp, err
	}

	if inventoryResp != nil && inventoryResp.StatusCode != 0 && inventoryResp.StatusCode != errno.StatusOK {
		l.Logger.Errorf("delete product inventory returned code: %d msg: %s", inventoryResp.StatusCode, inventoryResp.StatusMsg)
		resp.StatusCode = inventoryResp.StatusCode
		if inventoryResp.StatusMsg != "" {
			resp.StatusMsg = inventoryResp.StatusMsg
		} else {
			resp.StatusMsg = "delete inventory failed"
		}
		return resp, nil
	}

	if err := l.svcCtx.ProductModel.Delete(l.ctx, in.GetProductId()); err != nil {
		l.Logger.Errorf("delete product failed: %v", err)
		return resp, err
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	return resp, nil
}
