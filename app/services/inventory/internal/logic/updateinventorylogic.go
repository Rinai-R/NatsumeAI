package logic

import (
	"context"

	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/services/inventory/internal/svc"
	"NatsumeAI/app/services/inventory/inventory"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type UpdateInventoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateInventoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateInventoryLogic {
	return &UpdateInventoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 更新库存，商家调用
func (l *UpdateInventoryLogic) UpdateInventory(in *inventory.UpdateInventoryReq) (*inventory.InventoryResp, error) {
	resp := &inventory.InventoryResp{}
	if in == nil || in.GetMerchantId() <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid request payload"
		return resp, nil
	}

	merchantID := in.GetMerchantId()

	if len(in.Items) == 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "empty inventory items"
		return resp, nil
	}

	for _, item := range in.Items {
		if item == nil || item.ProductId <= 0 || item.Quantity == 0 {
			resp.StatusCode = errno.InvalidParam
			resp.StatusMsg = "invalid inventory item"
			return resp, nil
		}
	}

	err := l.svcCtx.InventoryModel.ExecWithTransaction(l.ctx, func(ctx context.Context, s sqlx.Session) error {
		for _, item := range in.Items {
			quantity := item.Quantity
			if quantity < 0 {
				if err := l.svcCtx.InventoryModel.DecrWithSessionByMerchant(ctx, s, item.ProductId, merchantID, -quantity); err != nil {
					l.Logger.Debug("rpc: 减少库存失败：", err, "冻结对象：", item)
					return err
				}
				continue
			}

			if err := l.svcCtx.InventoryModel.IncrWithSessionByMerchant(ctx, s, item.ProductId, merchantID, quantity); err != nil {
				l.Logger.Debug("rpc: 新增库存失败：", err, "冻结对象：", item)
				return err
			}
		}
		return nil
	})
	if err != nil {
		resp.StatusCode = errno.InternalError
		resp.StatusMsg = err.Error()
		return resp, nil
	}
	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	return resp, nil
}
