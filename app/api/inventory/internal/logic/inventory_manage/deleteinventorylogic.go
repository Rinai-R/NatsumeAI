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

type DeleteInventoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteInventoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteInventoryLogic {
	return &DeleteInventoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteInventoryLogic) DeleteInventory(req *types.DeleteInventoryRequest) (resp *types.InventoryActionResponse, err error) {
	if req == nil || req.ProductId <= 0 {
		return nil, errors.New(int(errno.InvalidParam), "invalid product id")
	}

	merchantID, err := util.UserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}

	in := &inventorysvc.DeleteInventoryReq{
		ProductId:  req.ProductId,
		MerchantId: merchantID,
	}

	res, err := l.svcCtx.InventoryRpc.DeleteInventory(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: delete inventory rpc failed: ", err)
		return nil, err
	}

	if res == nil {
		return nil, errors.New(int(errno.InternalError), "empty delete inventory response")
	}

	if res.StatusCode != errno.StatusOK {
		l.Logger.Error("logic: delete inventory rpc returned error: ", res)
		return nil, errors.New(int(res.StatusCode), res.StatusMsg)
	}


	resp = &types.InventoryActionResponse{
		StatusCode: res.StatusCode,
		StatusMsg:  res.StatusMsg,
	}

	return resp, nil
}
