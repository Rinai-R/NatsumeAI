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

type GetCartItemsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCartItemsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCartItemsLogic {
	return &GetCartItemsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCartItemsLogic) GetCartItems(req *types.GetCartItemsRequest) (resp *types.GetCartItemsResponse, err error) {
	userID, err := util.UserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}

	in := &cartsvc.GetCartItemListReq{
		UserId: userID,
	}

	res, err := l.svcCtx.CartRpc.GetCartItemList(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: get cart items rpc failed: ", err)
		return nil, err
	}
	if res == nil {
		return nil, errors.New(int(errno.InternalError), "empty get cart response")
	}

	items := make([]types.CartItem, 0, len(res.Data))
	for _, item := range res.Data {
		if item == nil {
			continue
		}
		items = append(items, types.CartItem{
			Id:        item.GetId(),
			UserId:    item.GetUserId(),
			ProductId: item.GetProductId(),
			Quantity:  item.GetQuantity(),
		})
	}

	resp = &types.GetCartItemsResponse{
		StatusCode: res.StatusCode,
		StatusMsg:  res.StatusMsg,
		Total:      res.Total,
		Items:      items,
	}

	return
}
