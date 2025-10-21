package helper

import (
	"NatsumeAI/app/api/inventory/internal/types"
	inventorysvc "NatsumeAI/app/services/inventory/inventoryservice"
)

func ToInventoryItem(item *inventorysvc.GetInventoryItem) types.InventoryItem {
	if item == nil {
		return types.InventoryItem{}
	}

	return types.InventoryItem{
		ProductId: item.ProductId,
		Inventory: item.Inventory,
		SoldCount: item.SoldCount,
	}
}
