package logic

import (
	"context"

	"NatsumeAI/app/common/consts/errno"
	inventoryModel "NatsumeAI/app/dal/inventory"
	"NatsumeAI/app/services/inventory/internal/svc"
	"NatsumeAI/app/services/inventory/inventory"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteInventoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteInventoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteInventoryLogic {
	return &DeleteInventoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 删除商品的时候同时调用这个
func (l *DeleteInventoryLogic) DeleteInventory(in *inventory.DeleteInventoryReq) (*inventory.InventoryResp, error) {
	resp := &inventory.InventoryResp{}

	if in == nil || in.GetProductId() <= 0 || in.GetMerchantId() <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid request payload"
		return resp, nil
	}

	record, err := l.svcCtx.InventoryModel.FindOneWithNoCache(l.ctx, in.GetProductId())
	if err != nil {
		if err == inventoryModel.ErrNotFound {
			// 保证幂等
			resp.StatusCode = errno.StatusOK
			resp.StatusMsg = "ok"
			return resp, nil
		}
		l.Logger.Errorf("delete inventory find failed: %v", err)
		return resp, err
	}

	if record.MerchantId != in.GetMerchantId() {
		resp.StatusCode = errno.MerchantMismatch
		resp.StatusMsg = "merchant mismatch"
		return resp, nil
	}

	if err := l.svcCtx.InventoryModel.Delete(l.ctx, in.GetProductId()); err != nil {
		l.Logger.Errorf("delete inventory failed: %v", err)
		return resp, err
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	return resp, nil
}
