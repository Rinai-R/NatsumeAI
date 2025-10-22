// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package cart

import (
	"context"

	"NatsumeAI/app/api/cart/internal/svc"
	"NatsumeAI/app/api/cart/internal/types"
	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/common/util"
	cartsvc "NatsumeAI/app/services/cart/cartservice"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/x/errors"
)

type AddCartItemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddCartItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddCartItemLogic {
	return &AddCartItemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddCartItemLogic) AddCartItem(req *types.AddCartItemRequest) (resp *types.CartActionResponse, err error) {
	if req == nil || req.ProductId <= 0 || req.Quantity <= 0 {
		return nil, errors.New(int(errno.InvalidParam), "invalid cart payload")
	}

	userID, err := util.UserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}

	in := &cartsvc.CreateCartItemReq{
		UserId:    userID,
		ProductId: req.ProductId,
		Count:     req.Quantity,
	}

	res, err := l.svcCtx.CartRpc.AddCartItem(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: add cart item rpc failed: ", err)
		return nil, err
	}
	if res == nil {
		return nil, errors.New(int(errno.InternalError), "empty add cart response")
	}

	resp = &types.CartActionResponse{
		StatusCode: res.StatusCode,
		StatusMsg:  res.StatusMsg,
	}

	return
}
