package logic

import (
	"context"

	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/services/inventory/internal/svc"
	"NatsumeAI/app/services/inventory/inventory"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type DecreasePreInventoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDecreasePreInventoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DecreasePreInventoryLogic {
	return &DecreasePreInventoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 预扣
func (l *DecreasePreInventoryLogic) DecreasePreInventory(in *inventory.InventoryReq) (*inventory.InventoryResp, error) {
	resp := &inventory.InventoryResp{}
	err := l.svcCtx.InventoryModel.ExecWithTransaction(l.ctx, func(ctx context.Context, s sqlx.Session) error {
		for _, item := range in.Items {
			err := l.svcCtx.InventoryModel.
				FreezeWithSession(ctx, s, item.ProductId, item.Quantity)
			if err != nil {
				l.Logger.Debug("rpc: 冻结库存失败：", err, "冻结对象：", item)
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
