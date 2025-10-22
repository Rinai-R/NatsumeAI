package logic

import (
	"context"

	"NatsumeAI/app/common/consts/errno"
	cartmodel "NatsumeAI/app/dal/cart"
	cartpb "NatsumeAI/app/services/cart/cart"
	"NatsumeAI/app/services/cart/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddCartItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddCartItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddCartItemLogic {
	return &AddCartItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 添加到购物车
func (l *AddCartItemLogic) AddCartItem(in *cartpb.CreateCartItemReq) (*cartpb.CartItemResp, error) {
	resp := &cartpb.CartItemResp{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
	}

	if in == nil || in.UserId <= 0 || in.ProductId <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid request payload"
		return resp, nil
	}

	_, err := l.svcCtx.CartModel.FindOneByUserProduct(l.ctx, in.UserId, in.ProductId)
	switch err {
	case nil:
		l.Logger.Errorf("item exists")
		resp.StatusCode = errno.ItemExistInCart
		resp.StatusMsg = "item exist in cart"
		return resp, nil
	case cartmodel.ErrNotFound:
		_, err = l.svcCtx.CartModel.Insert(l.ctx, &cartmodel.Cart{
			UserId:    in.UserId,
			ProductId: in.ProductId,
			Quantity:  in.Count,
		})
		if err != nil {
			l.Logger.Errorf("add cart item insert failed: %v", err)
			return resp, err
		}
	default:
		return resp, err
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	return resp, nil
}
