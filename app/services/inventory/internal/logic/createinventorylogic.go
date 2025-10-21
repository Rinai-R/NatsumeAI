package logic

import (
	"context"
	"fmt"

	"NatsumeAI/app/common/consts/errno"
	inventoryModel "NatsumeAI/app/dal/inventory"
	"NatsumeAI/app/services/inventory/internal/svc"
	"NatsumeAI/app/services/inventory/inventory"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateInventoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateInventoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateInventoryLogic {
	return &CreateInventoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 为商品创建库存
func (l *CreateInventoryLogic) CreateInventory(in *inventory.CreateInventoryReq) (*inventory.InventoryResp, error) {
	resp := &inventory.InventoryResp{}
	if in == nil || in.GetProductId() <= 0 || in.GetMerchantId() <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid request payload"
		return resp, nil
	}

	_, err := l.svcCtx.InventoryModel.Insert(l.ctx, &inventoryModel.Inventory{
		ProductId:  in.ProductId,
		MerchantId: in.MerchantId,
		Stock:      in.Inventory,
	})
	if err != nil {
		resp.StatusCode = errno.InsertInventoryError
		resp.StatusMsg = fmt.Sprint("rpc： inventory 插入数据失败", err.Error())
	}
	return resp, nil
}
