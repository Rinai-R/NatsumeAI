package logic

import (
	"context"

	"NatsumeAI/app/api/user/internal/types"
	"NatsumeAI/app/common/consts/biz"
	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/services/user/userservice"

	"github.com/zeromicro/x/errors"
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

func userIDFromCtx(ctx context.Context) (int64, error) {
	if ctx == nil {
		return 0, errors.New(int(errno.TokenEmpty), "missing context")
	}

	if v := ctx.Value(biz.USER_KEY); v != nil {
		switch val := v.(type) {
		case int64:
			return val, nil
		case uint64:
			return int64(val), nil
		}
	}

	return 0, errors.New(int(errno.TokenEmpty), "unauthorized")
}
