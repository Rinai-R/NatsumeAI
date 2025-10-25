package logic

import (
	"context"
	"errors"
	"fmt"

	"NatsumeAI/app/common/consts/errno"
	inventorymodel "NatsumeAI/app/dal/inventory"
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

// 预扣（单商品）
func (l *DecreasePreInventoryLogic) DecreasePreInventory(in *inventory.InventoryReq) (*inventory.InventoryResp, error) {
    resp := &inventory.InventoryResp{}

    if in == nil || in.OrderId <= 0 || in.Item == nil {
        resp.StatusCode = errno.InvalidParam
        resp.StatusMsg = "invalid order or item"
        return resp, nil
    }

    ticket, err := l.svcCtx.InventoryTokenModel.CheckToken(l.ctx, in.OrderId, false)
	if err != nil {
		var tokenErr *inventorymodel.TokenError
		switch {
		case errors.As(err, &tokenErr):
			resp.StatusCode = errno.InvalidParam
			resp.StatusMsg = tokenErr.Error()
			l.Logger.Infof("pre inventory token check rejected: order=%d code=%s details=%v", in.OrderId, tokenErr.Code(), tokenErr.Details())
		default:
			resp.StatusCode = errno.InternalError
			resp.StatusMsg = err.Error()
			l.Logger.Errorf("pre inventory token check failed: order=%d err=%v", in.OrderId, err)
		}
		return resp, nil
	}

    if err := matchTicketAndRequest(ticket, in.Item); err != nil {
        resp.StatusCode = errno.InvalidParam
        resp.StatusMsg = err.Error()
        l.Logger.Infof("pre inventory token mismatch: order=%d err=%v", in.OrderId, err)
        return resp, nil
    }

    err = l.svcCtx.InventoryModel.ExecWithTransaction(l.ctx, func(ctx context.Context, s sqlx.Session) error {
        item := in.Item
        err := l.svcCtx.InventoryModel.
            FreezeWithSession(ctx, s, item.ProductId, item.Quantity)
        if err != nil {
            l.Logger.Debug("rpc: 冻结库存失败：", err, "冻结对象：", item)
            return err
        }
        _, err = l.svcCtx.InventoryAuditModel.InsertWithSession(l.ctx, s, &inventorymodel.InventoryAudit{
            OrderId:   in.OrderId,
            ProductId: item.ProductId,
            Quantity:  item.Quantity,
            Status:    inventorymodel.AUDIT_PENDING,
        })
        if err != nil {
            l.Logger.Debug("rpc: 冻结库存插入审计日志失败：", err, "冻结对象：", item)
            return err
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

func matchTicketAndRequest(ticket *inventorymodel.TokenTicket, item *inventory.Item) error {
    if ticket == nil {
        return fmt.Errorf("token ticket missing")
    }
    if item == nil || item.ProductId <= 0 || item.Quantity <= 0 {
        return fmt.Errorf("invalid request item")
    }
    if ticket.Item.SKU != item.ProductId {
        return fmt.Errorf("token missing sku %d", item.ProductId)
    }
    if ticket.Item.Quantity < item.Quantity {
        return fmt.Errorf("not enough token for sku %d need %d have %d", item.ProductId, item.Quantity, ticket.Item.Quantity)
    }
    return nil
}
