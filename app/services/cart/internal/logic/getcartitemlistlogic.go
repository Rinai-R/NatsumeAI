package logic

import (
	"context"

	"NatsumeAI/app/common/consts/errno"
	cartmodel "NatsumeAI/app/dal/cart"
	cartpb "NatsumeAI/app/services/cart/cart"
	"NatsumeAI/app/services/cart/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCartItemListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetCartItemListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCartItemListLogic {
	return &GetCartItemListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 用户购物车信息
func (l *GetCartItemListLogic) GetCartItemList(in *cartpb.GetCartItemListReq) (*cartpb.GetCartItemListResp, error) {
	resp := &cartpb.GetCartItemListResp{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
	}

	if in == nil || in.UserId <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid request payload"
		return resp, nil
	}

	items, err := l.svcCtx.CartModel.ListByUserId(l.ctx, in.UserId)
	if err != nil {
		if err != cartmodel.ErrNotFound {
			return resp, err
		}
		items = nil
	}

	respItems := make([]*cartpb.CartInfoResp, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		respItems = append(respItems, &cartpb.CartInfoResp{
			Id:        item.Id,
			UserId:    item.UserId,
			ProductId: item.ProductId,
			Quantity:  item.Quantity,
		})
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	resp.Total = int64(len(respItems))
	resp.Data = respItems

	return resp, nil
}
