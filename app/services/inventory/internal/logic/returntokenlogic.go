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

    if len(in.Item) == 0 {
        resp.StatusCode = errno.InvalidParam
        resp.StatusMsg = "empty items"
        return resp, nil
    }

    items := make([]inventorymodel.TokenItem, 0, len(in.Item))
    for _, item := range in.Item {
        if item.ProductId <= 0 || item.Quantity <= 0 {
            resp.StatusCode = errno.InvalidParam
            resp.StatusMsg = fmt.Sprintf("invalid item productId=%d quantity=%d", item.ProductId, item.Quantity)
            return resp, nil
        }
        items = append(items, inventorymodel.TokenItem{SKU: item.ProductId, Quantity: item.Quantity})
    }

	// optional: verify ticket snapshot if exists
	if ticket, err := l.svcCtx.InventoryTokenModel.CheckToken(l.ctx, in.PreorderId, false); err != nil {
		var tokenErr *inventorymodel.TokenError
		if errors.As(err, &tokenErr) {
			if tokenErr.Code() != "TICKET_NOT_FOUND" {
				resp.StatusCode = errno.InvalidParam
				resp.StatusMsg = tokenErr.Error()
				l.Logger.Infof("return token check rejected: preorder=%d code=%s details=%v", in.PreorderId, tokenErr.Code(), tokenErr.Details())
				return resp, nil
			}
		} else {
			resp.StatusCode = errno.InternalError
			resp.StatusMsg = err.Error()
			return resp, nil
		}
    } else if err := ensureReturnItemsMatch(items, ticket.Item); err != nil {
        resp.StatusCode = errno.InvalidParam
        resp.StatusMsg = err.Error()
        return resp, nil
    }

    for _, it := range items {
        err := l.svcCtx.InventoryTokenModel.ReturnToken(l.ctx, in.PreorderId, it)
        if err != nil {
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

func ensureReturnItemsMatch(req []inventorymodel.TokenItem, ticketItem inventorymodel.TokenItem) error {
    if ticketItem.SKU == 0 || ticketItem.Quantity <= 0 {
        return nil
    }
    // 单商品：允许部分归还（比如取消部分数量），只校验 SKU 一致
    for _, item := range req {
        if item.SKU != ticketItem.SKU {
            return fmt.Errorf("unexpected sku %d, expect %d", item.SKU, ticketItem.SKU)
        }
    }
    return nil
}
