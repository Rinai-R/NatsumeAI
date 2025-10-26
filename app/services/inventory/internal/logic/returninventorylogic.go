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

// 归还库存（单商品）
func (l *ReturnInventoryLogic) ReturnInventory(in *inventory.InventoryReq) (*inventory.InventoryResp, error) {
    resp := &inventory.InventoryResp{}

    var tokenItem *inventorymodel.TokenItem
    // 优先使用 preorderId，其次使用 orderId（系统中二者相同，但容错更稳妥）
    var pid int64
    if in != nil {
        if in.PreorderId > 0 {
            pid = in.PreorderId
        } else if in.OrderId > 0 {
            pid = in.OrderId
        }
    }
    if pid > 0 {
        // 仅查询，不消费；待 DB 回滚成功后再归还并删除 ticket
        if ticket, err := l.svcCtx.InventoryTokenModel.CheckToken(l.ctx, pid, false); err != nil {
            var tokenErr *inventorymodel.TokenError
            if errors.As(err, &tokenErr) {
                if tokenErr.Code() == "TICKET_NOT_FOUND" {
                    l.Logger.Infof("return inventory token ticket missing: pre_order=%d", pid)
                } else {
                    l.Logger.Infof("return inventory token check warning: pre_order=%d code=%s details=%v", pid, tokenErr.Code(), tokenErr.Details())
                }
            } else {
                l.Logger.Errorf("return inventory token check failed: pre_order=%d err=%v", pid, err)
            }
        } else if ticket != nil {
            t := ticket.Item
            tokenItem = &t
        }
    }

    if in == nil || in.Item == nil || in.Item.ProductId <= 0 || in.Item.Quantity <= 0 {
        resp.StatusCode = errno.InvalidParam
        resp.StatusMsg = "invalid order or item"
        return resp, nil
    }

    err := l.svcCtx.InventoryModel.ExecWithTransaction(l.ctx, func(ctx context.Context, s sqlx.Session) error {
        item := in.Item
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
        return nil
    })
	if err != nil {
		resp.StatusCode = errno.InternalError
		resp.StatusMsg = err.Error()
		return resp, nil
	}

    if tokenItem != nil && pid > 0 {
        if tokenErr := l.svcCtx.InventoryTokenModel.ReturnToken(l.ctx, pid, *tokenItem); tokenErr != nil {
            l.Logger.Errorf("return inventory token release failed: preorder=%d err=%v item=%+v", pid, tokenErr, *tokenItem)
        } else {
            l.Logger.Infof("return inventory token released: preorder=%d", pid)
        }
    }

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	return resp, nil
}
