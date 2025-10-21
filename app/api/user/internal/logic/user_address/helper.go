package logic

import (
	"NatsumeAI/app/api/user/internal/types"
	"NatsumeAI/app/services/user/userservice"
)

func toAddress(addr *userservice.Address) types.Address {
	if addr == nil {
		return types.Address{}
	}

	return types.Address{
		AddressId: addr.AddressId,
		Detail:    addr.Detail,
		IsDefault: addr.IsDefault,
		CreatedAt: addr.CreatedAt,
		UpdatedAt: addr.UpdatedAt,
	}
}

func toAddressSlice(addrs []*userservice.Address) []types.Address {
	if len(addrs) == 0 {
		return nil
	}

	resp := make([]types.Address, 0, len(addrs))
	for _, addr := range addrs {
		resp = append(resp, toAddress(addr))
	}

	return resp
}
