package logic

import (
	"context"

	"NatsumeAI/app/common/consts/errno"
	cartmodel "NatsumeAI/app/dal/cart"
	cartpb "NatsumeAI/app/services/cart/cart"
	"NatsumeAI/app/services/cart/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCartItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateCartItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCartItemLogic {
	return &UpdateCartItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 改变商品数量
func (l *UpdateCartItemLogic) UpdateCartItem(in *cartpb.UpdateCartItemReq) (*cartpb.CartItemResp, error) {
	resp := &cartpb.CartItemResp{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
	}

	if in == nil || in.UserId <= 0 || in.ProductId <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid request payload"
		return resp, nil
	}

	item, err := l.svcCtx.CartModel.FindOneByUserProduct(l.ctx, in.UserId, in.ProductId)
	if err != nil {
		if err == cartmodel.ErrNotFound {
			resp.StatusCode = errno.CartItemNotFound
			resp.StatusMsg = "cart item not found"
			return resp, nil
		}
		return resp, err
	}

	if item.Quantity + in.Count <= 0 {
		if err := l.svcCtx.CartModel.Delete(l.ctx, item.Id); err != nil {
			l.Logger.Errorf("update cart item delete failed: %v", err)
			return resp, err
		}
		resp.StatusCode = errno.StatusOK
		resp.StatusMsg = "ok"
		return resp, nil
	}

	item.Quantity += in.Count
	if err := l.svcCtx.CartModel.Update(l.ctx, item); err != nil {
		l.Logger.Errorf("update cart item failed: %v", err)
		return resp, err
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	return resp, nil
}
