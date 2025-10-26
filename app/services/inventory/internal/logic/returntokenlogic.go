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
)

type ReturnTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReturnTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReturnTokenLogic {
	return &ReturnTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 归还令牌
func (l *ReturnTokenLogic) ReturnToken(in *inventory.ReturnTokenReq) (*inventory.InventoryResp, error) {
    resp := &inventory.InventoryResp{}

    if in == nil || in.PreorderId <= 0 {
        resp.StatusCode = errno.InvalidParam
        resp.StatusMsg = "invalid preorder id"
        return resp, nil
    }

    item := in.GetItem()
    if item == nil || item.ProductId <= 0 || item.Quantity <= 0 {
        resp.StatusCode = errno.InvalidParam
        resp.StatusMsg = fmt.Sprintf("invalid item productId=%d quantity=%d", item.GetProductId(), item.GetQuantity())
        return resp, nil
    }

    // Check ticket to obtain epoch; if not found, treat as idempotent success.
    ticket, err := l.svcCtx.InventoryTokenModel.CheckToken(l.ctx, in.PreorderId, false)
    if err != nil {
        var tokenErr *inventorymodel.TokenError
        if errors.As(err, &tokenErr) {
            if tokenErr.Code() == "TICKET_NOT_FOUND" {
                l.Logger.Infof("return token ticket missing: preorder=%d", in.PreorderId)
                resp.StatusCode = errno.StatusOK
                resp.StatusMsg = "ok"
                return resp, nil
            }
            resp.StatusCode = errno.InvalidParam
            resp.StatusMsg = tokenErr.Error()
            l.Logger.Infof("return token check rejected: preorder=%d code=%s details=%v", in.PreorderId, tokenErr.Code(), tokenErr.Details())
            return resp, nil
        }
        resp.StatusCode = errno.InternalError
        resp.StatusMsg = err.Error()
        return resp, nil
    }

    // Validate SKU against ticket and build token item with epoch
    if ticket != nil {
        if err := ensureReturnItemMatch(item.GetProductId(), ticket.Item); err != nil {
            resp.StatusCode = errno.InvalidParam
            resp.StatusMsg = err.Error()
            return resp, nil
        }
        tokenItem := inventorymodel.TokenItem{
            SKU:      item.ProductId,
            Quantity: item.Quantity,
            Epoch:    ticket.Item.Epoch,
        }
        if err := l.svcCtx.InventoryTokenModel.ReturnToken(l.ctx, in.PreorderId, tokenItem); err != nil {
            var tokenErr *inventorymodel.TokenError
            if errors.As(err, &tokenErr) {
                resp.StatusCode = errno.InvalidParam
                resp.StatusMsg = tokenErr.Error()
                l.Logger.Infof("return token rejected: preorder=%d code=%s details=%v", in.PreorderId, tokenErr.Code(), tokenErr.Details())
                return resp, nil
            }
            resp.StatusCode = errno.InternalError
            resp.StatusMsg = err.Error()
            return resp, nil
        }
    }

    resp.StatusCode = errno.StatusOK
    resp.StatusMsg = "ok"
    return resp, nil
}

func ensureReturnItemMatch(reqSKU int64, ticketItem inventorymodel.TokenItem) error {
    if ticketItem.SKU == 0 || ticketItem.Quantity <= 0 {
        return nil
    }
    if reqSKU != ticketItem.SKU {
        return fmt.Errorf("unexpected sku %d, expect %d", reqSKU, ticketItem.SKU)
    }
    return nil
}
