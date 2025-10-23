package logic

import (
	"context"
	"errors"

	"NatsumeAI/app/common/consts/errno"
	inventorymodel "NatsumeAI/app/dal/inventory"
	"NatsumeAI/app/services/inventory/internal/svc"
	"NatsumeAI/app/services/inventory/inventory"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ReturnInventoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReturnInventoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReturnInventoryLogic {
	return &ReturnInventoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 归还库存
func (l *ReturnInventoryLogic) ReturnInventory(in *inventory.InventoryReq) (*inventory.InventoryResp, error) {
	resp := &inventory.InventoryResp{}

	var tokenItems []inventorymodel.TokenItem
	if in != nil && in.OrderId > 0 {
		if ticket, err := l.svcCtx.InventoryTokenModel.CheckToken(l.ctx, in.OrderId, true); err != nil {
			var tokenErr *inventorymodel.TokenError
			if errors.As(err, &tokenErr) {
				if tokenErr.Code() == "TICKET_NOT_FOUND" {
					l.Logger.Infof("return inventory token ticket missing: order=%d", in.OrderId)
				} else {
					l.Logger.Infof("return inventory token check warning: order=%d code=%s details=%v", in.OrderId, tokenErr.Code(), tokenErr.Details())
				}
			} else {
				l.Logger.Errorf("return inventory token check failed: order=%d err=%v", in.OrderId, err)
			}
		} else if ticket != nil {
			tokenItems = append(tokenItems, ticket.Items...)
		}
	}

	err := l.svcCtx.InventoryModel.ExecWithTransaction(l.ctx, func(ctx context.Context, s sqlx.Session) error {
		for _, item := range in.Items {
			err := l.svcCtx.InventoryModel.
				CancleSoldWithSession(ctx, s, item.ProductId, item.Quantity)
			if err != nil {
				l.Logger.Debug("rpc: 取消库存扣减失败：", err, "冻结对象：", item)
				return err
			}
			err = l.svcCtx.InventoryAuditModel.UpdateStatusWithSession(ctx, s, in.OrderId, item.ProductId, inventorymodel.AUDIT_CANCLLED)
			if err != nil {
				l.Logger.Debug("rpc: 取消库存扣减审计日志失败：", err, "冻结对象：", item)
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

	if len(tokenItems) > 0 {
		if restored, skipped, tokenErr := l.svcCtx.InventoryTokenModel.ReturnToken(l.ctx, in.OrderId, tokenItems); tokenErr != nil {
			l.Logger.Errorf("return inventory token release failed: order=%d err=%v", in.OrderId, tokenErr)
		} else {
			l.Logger.Infof("return inventory token released: order=%d restored=%d skipped=%d", in.OrderId, restored, skipped)
		}
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	return resp, nil
}
