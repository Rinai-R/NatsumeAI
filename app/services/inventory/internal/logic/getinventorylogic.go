package logic

import (
	"context"

	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/services/inventory/internal/svc"
	"NatsumeAI/app/services/inventory/inventory"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetInventoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetInventoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetInventoryLogic {
	return &GetInventoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取库存
func (l *GetInventoryLogic) GetInventory(in *inventory.GetInventoryReq) (*inventory.GetInventoryResp, error) {
	// todo: add your logic here and delete this line
	resp := &inventory.GetInventoryResp{}
	items := make([]*inventory.GetInventoryItem, 0, len(in.ProductIds))

	for _, id := range in.ProductIds {
		product, err := l.svcCtx.InventoryModel.FindOneWithNoCache(l.ctx, id)
		l.Logger.Error(product, err)
		if err != nil {
			items = append(items, &inventory.GetInventoryItem{
				ProductId: id,
			})
			l.Logger.Error("Get Inventory Item Failed: ", err)
			continue
		}
		items = append(items, &inventory.GetInventoryItem{
			ProductId: product.ProductId,
			Inventory: product.Stock,
			SoldCount: product.Sold,
		})
	}
	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	resp.Items = items
	return resp, nil
}
