package logic

import (
	"context"

	"NatsumeAI/app/common/consts/errno"
	cartmodel "NatsumeAI/app/dal/cart"
	cartpb "NatsumeAI/app/services/cart/cart"
	"NatsumeAI/app/services/cart/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteCartItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteCartItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCartItemLogic {
	return &DeleteCartItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 删了
func (l *DeleteCartItemLogic) DeleteCartItem(in *cartpb.DelCartItemReq) (*cartpb.CartItemResp, error) {
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

	if err := l.svcCtx.CartModel.Delete(l.ctx, item.Id); err != nil {
		l.Logger.Errorf("delete cart item failed: %v", err)
		return resp, err
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	return resp, nil
}
