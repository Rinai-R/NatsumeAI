package logic

import (
	"context"

	model "NatsumeAI/app/dal/user"
	"NatsumeAI/app/services/user/user"
)

func userToProfile(u *model.Users) *user.UserProfile {
	if u == nil {
		return nil
	}
	return &user.UserProfile{
		UserId:    int64(u.Id),
		Username:  u.Username,
		CreatedAt: u.CreatedAt.Unix(),
		UpdatedAt: u.UpdatedAt.Unix(),
	}
}

func addressToProto(addr *model.UserAddresses) *user.Address {
	if addr == nil {
		return nil
	}
	return &user.Address{
		AddressId: int64(addr.Id),
		UserId:    int64(addr.UserId),
		Detail:    addr.Detail,
		IsDefault: addr.IsDefault != 0,
		CreatedAt: addr.CreatedAt.Unix(),
		UpdatedAt: addr.UpdatedAt.Unix(),
	}
}

func boolToInt64(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

func addressesToProto(addrs []*model.UserAddresses) []*user.Address {
	resp := make([]*user.Address, 0, len(addrs))
	if len(addrs) == 0 {
		return resp
	}
	for _, addr := range addrs {
		resp = append(resp, addressToProto(addr))
	}
	return resp
}

func ensureDefaultForUser(ctx context.Context, addressModel model.UserAddressesModel, userId uint64, keepID uint64) error {
	addresses, err := addressModel.FindByUserId(ctx, userId)
	if err != nil {
		if err == model.ErrNotFound {
			return nil
		}
		return err
	}

	for _, addr := range addresses {
		desired := int64(0)
		if addr.Id == keepID {
			desired = 1
		}
		if addr.IsDefault == desired {
			continue
		}
		addr.IsDefault = desired
		if err := addressModel.Update(ctx, addr); err != nil {
			return err
		}
	}

	return nil
}
