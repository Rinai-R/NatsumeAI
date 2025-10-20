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
	err := l.svcCtx.InventoryModel.ExecWithTransaction(l.ctx, func(ctx context.Context, s sqlx.Session) error {
		for _, item := range in.Items {
			if item.Quantity < 0 {
				// 小于零表示扣减
				item.Quantity = -item.Quantity
				err := l.svcCtx.InventoryModel.
					DecrWithSession(ctx, s, item.ProductId, item.Quantity)
				if err != nil {
					l.Logger.Debug("rpc: 减少库存失败：", err, "冻结对象：", item)
					return err
				}
			} else {
				// 大于零表示新增库存
				err := l.svcCtx.InventoryModel.
					IncrWithSession(ctx, s, item.ProductId, item.Quantity)
				if err != nil {
					l.Logger.Debug("rpc: 新增库存失败：", err, "冻结对象：", item)
					return err
				}
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
