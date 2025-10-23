package logic

import (
	"context"

	"NatsumeAI/app/common/consts/errno"
	inventorymodel "NatsumeAI/app/dal/inventory"
	"NatsumeAI/app/services/inventory/internal/svc"
	"NatsumeAI/app/services/inventory/inventory"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type DecreaseInventoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDecreaseInventoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DecreaseInventoryLogic {
	return &DecreaseInventoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 实际扣减
func (l *DecreaseInventoryLogic) DecreaseInventory(in *inventory.InventoryReq) (*inventory.InventoryResp, error) {
	// todo: add your logic here and delete this line
	resp := &inventory.InventoryResp{}
	err := l.svcCtx.InventoryModel.ExecWithTransaction(l.ctx, func(ctx context.Context, s sqlx.Session) error {
		for _, item := range in.Items {
			err := l.svcCtx.InventoryModel.
				ConfirmWithSession(ctx, s, item.ProductId, item.Quantity)
			if err != nil {
				l.Logger.Debug("rpc: 扣减库存失败：", err, "冻结对象：", item)
				return err
			}
			err = l.svcCtx.InventoryAuditModel.UpdateStatusWithSession(ctx, s, in.OrderId, item.ProductId, inventorymodel.AUDIT_CONFIRMED)
			if err != nil {
				l.Logger.Debug("rpc: 扣减库存记录审计日志：", err, "冻结对象：", item)
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
