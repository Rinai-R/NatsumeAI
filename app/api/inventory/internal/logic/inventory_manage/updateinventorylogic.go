// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package inventory_manage

import (
	"context"

	"NatsumeAI/app/api/inventory/internal/svc"
	"NatsumeAI/app/api/inventory/internal/types"
	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/common/util"
	inventorysvc "NatsumeAI/app/services/inventory/inventoryservice"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/x/errors"
)

type UpdateInventoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateInventoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateInventoryLogic {
	return &UpdateInventoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateInventoryLogic) UpdateInventory(req *types.UpdateInventoryRequest) (resp *types.InventoryActionResponse, err error) {
	if req == nil {
		return nil, errors.New(int(errno.InvalidParam), "missing request payload")
	}

	userId, err := util.UserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}

	item := req.Items
	if item.ProductId <= 0 || item.Quantity == 0 {
		return nil, errors.New(int(errno.InvalidParam), "invalid inventory item")
	}

	in := &inventorysvc.UpdateInventoryReq{
		MerchantId: userId,
		Item: &inventorysvc.Item{
			ProductId: item.ProductId,
			Quantity:  item.Quantity,
		},
	}

	res, err := l.svcCtx.InventoryRpc.UpdateInventory(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: update inventory rpc failed: ", err)
		return nil, err
	}

	if res == nil {
		return nil, errors.New(int(errno.InternalError), "empty update inventory response")
	}

	if res.StatusCode != errno.StatusOK {
		l.Logger.Error("logic: update inventory rpc returned error: ", res)
		return nil, errors.New(int(res.StatusCode), res.StatusMsg)
	}

	resp = &types.InventoryActionResponse{
		StatusCode: res.StatusCode,
		StatusMsg:  res.StatusMsg,
	}

	return resp, nil
}
