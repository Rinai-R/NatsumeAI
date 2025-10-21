// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package inventory_manage

import (
	"context"

	"NatsumeAI/app/api/inventory/internal/logic/helper"
	"NatsumeAI/app/api/inventory/internal/svc"
	"NatsumeAI/app/api/inventory/internal/types"
	"NatsumeAI/app/common/consts/errno"
	inventorysvc "NatsumeAI/app/services/inventory/inventoryservice"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/x/errors"
)

type GetInventoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetInventoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetInventoryLogic {
	return &GetInventoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetInventoryLogic) GetInventory(req *types.GetInventoryRequest) (resp *types.GetInventoryResponse, err error) {
	if req == nil || req.ProductId <= 0 {
		return nil, errors.New(int(errno.InvalidParam), "invalid product id")
	}

	in := &inventorysvc.GetInventoryReq{
		ProductIds: []int64{req.ProductId},
	}

	res, err := l.svcCtx.InventoryRpc.GetInventory(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: get inventory rpc failed: ", err)
		return nil, err
	}

	if res == nil {
		return nil, errors.New(int(errno.InternalError), "empty get inventory response")
	}

	if res.StatusCode != errno.StatusOK {
		l.Logger.Error("logic: get inventory rpc returned error: ", res)
		return nil, errors.New(int(res.StatusCode), res.StatusMsg)
	}

	item := types.InventoryItem{}
	if len(res.Items) > 0 {
		item = helper.ToInventoryItem(res.Items[0])
	}

	resp = &types.GetInventoryResponse{
		StatusCode: res.StatusCode,
		StatusMsg: res.StatusMsg,
		Items: item,
	}

	return resp, nil
}
