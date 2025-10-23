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

// 结账的时候，根据库存快速发放令牌
func (l *TryGetTokenLogic) TryGetToken(in *inventory.TryGetTokenReq) (*inventory.InventoryResp, error) {
	resp := &inventory.InventoryResp{}

	if in == nil || in.PreorderId <= 0 || len(in.Items) == 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid preorder or items"
		return resp, nil
	}

	skus := make([]int64, 0, len(in.Items))
	tokenItems := make([]inventorymodel.TokenItem, 0, len(in.Items))
	for _, item := range in.Items {
		skus = append(skus, item.ProductId)
		tokenItems = append(tokenItems, inventorymodel.TokenItem{
			SKU:      item.ProductId,
			Quantity: item.Quantity,
		})
	}

	if err := l.svcCtx.InventoryTokenModel.SyncTokenSnapshots(l.ctx, skus); err != nil {
		var tokenErr *inventorymodel.TokenError
		switch {
		case errors.As(err, &tokenErr):
			code := errno.InvalidParam
			if tokenErr.Code() == "SKU_NOT_FOUND" {
				code = errno.ProductNotFound
			}
			resp.StatusCode = int32(code)
			resp.StatusMsg = tokenErr.Error()
			l.Logger.Infof("inventory token sync rejected: preorder=%d code=%s details=%v", in.PreorderId, tokenErr.Code(), tokenErr.Details())
		default:
			resp.StatusCode = errno.InternalError
			resp.StatusMsg = err.Error()
			l.Logger.Errorf("inventory token sync failed: preorder=%d err=%v", in.PreorderId, err)
		}
		return resp, nil
	}

	if err := l.svcCtx.InventoryTokenModel.TryGetToken(l.ctx, in.PreorderId, tokenItems); err != nil {
		var tokenErr *inventorymodel.TokenError
		switch {
		case errors.As(err, &tokenErr):
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
	l.Logger.Infof("inventory token granted: preorder=%d items=%d", in.PreorderId, len(in.Items))
	return resp, nil
}
