package logic

import (
	"context"

	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/services/inventory/internal/svc"
	"NatsumeAI/app/services/inventory/inventory"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ReturnPreInventoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReturnPreInventoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReturnPreInventoryLogic {
	return &ReturnPreInventoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 归还预扣
func (l *ReturnPreInventoryLogic) ReturnPreInventory(in *inventory.InventoryReq) (*inventory.InventoryResp, error) {
	resp := &inventory.InventoryResp{}
	err := l.svcCtx.InventoryModel.ExecWithTransaction(l.ctx, func(ctx context.Context, s sqlx.Session) error {
		for _, item := range in.Items {
			err := l.svcCtx.InventoryModel.
				UnfreezeWithSession(ctx, s, item.ProductId, item.Quantity)
			if err != nil {
				l.Logger.Debug("rpc: 解冻库存失败：", err, "冻结对象：", item)
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
