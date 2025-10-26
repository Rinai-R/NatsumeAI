package logic

import (
	"context"
	"errors"

	"NatsumeAI/app/common/consts/errno"
	inventorymodel "NatsumeAI/app/dal/inventory"
	"NatsumeAI/app/services/inventory/internal/svc"
	"NatsumeAI/app/services/inventory/inventory"

	"github.com/zeromicro/go-zero/core/logx"
)

type TryGetTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTryGetTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TryGetTokenLogic {
	return &TryGetTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 结账的时候，根据库存快速发放令牌（单商品）
func (l *TryGetTokenLogic) TryGetToken(in *inventory.TryGetTokenReq) (*inventory.InventoryResp, error) {
    resp := &inventory.InventoryResp{}

    if in == nil || in.PreorderId <= 0 || in.Item == nil {
        resp.StatusCode = errno.InvalidParam
        resp.StatusMsg = "invalid preorder or item"
        return resp, nil
    }

    if in.Item.ProductId <= 0 || in.Item.Quantity <= 0 {
        resp.StatusCode = errno.InvalidParam
        resp.StatusMsg = "invalid item"
        return resp, nil
    }

    tokenItem := inventorymodel.TokenItem{
        SKU:      in.Item.ProductId,
        Quantity: in.Item.Quantity,
    }

    if err := l.svcCtx.InventoryTokenModel.TryGetToken(l.ctx, in.PreorderId, tokenItem); err != nil {
        var tokenErr *inventorymodel.TokenError
        switch {
		case errors.As(err, &tokenErr):
			// 幂等：重复请求直接视为成功
			if tokenErr.Code() == "DUPLICATE" {
				resp.StatusCode = errno.StatusOK
				resp.StatusMsg = "ok"
				l.Logger.Infof("inventory token duplicate treated as ok: preorder=%d sku=%d", in.PreorderId, in.Item.ProductId)
				return resp, nil
			}
			code := errno.InvalidParam
			if tokenErr.Code() == "SKU_NOT_FOUND" {
				code = errno.ProductNotFound
			}
			resp.StatusCode = int32(code)
			resp.StatusMsg = tokenErr.Error()
			l.Logger.Infof("inventory token rejected: preorder=%d code=%s details=%v", in.PreorderId, tokenErr.Code(), tokenErr.Details())
		default:
			resp.StatusCode = errno.InternalError
			resp.StatusMsg = err.Error()
			l.Logger.Errorf("inventory token failed: preorder=%d err=%v", in.PreorderId, err)
		}
		return resp, nil
	}

    resp.StatusCode = errno.StatusOK
    resp.StatusMsg = "ok"
    l.Logger.Infof("inventory token granted: preorder=%d sku=%d qty=%d", in.PreorderId, in.Item.ProductId, in.Item.Quantity)
    return resp, nil
}
