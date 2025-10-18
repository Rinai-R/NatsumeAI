package logic

import (
	"context"
	"strings"

	"NatsumeAI/app/common/consts/errno"
	model "NatsumeAI/app/dal/user"
	"NatsumeAI/app/services/user/internal/svc"
	"NatsumeAI/app/services/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateAddressLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateAddressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAddressLogic {
	return &UpdateAddressLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateAddressLogic) UpdateAddress(in *user.UpdateAddressRequest) (*user.UpdateAddressResponse, error) {
	resp := &user.UpdateAddressResponse{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
	}

	if in == nil || in.Address == nil {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "address payload is required"
		return resp, nil
	}

	if in.UserId <= 0 || in.AddressId <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid identifier"
		return resp, nil
	}

	addr, err := l.svcCtx.UserAddressModel.FindOne(l.ctx, uint64(in.AddressId))
	if err != nil {
		if err == model.ErrNotFound {
			resp.StatusCode = errno.AddressNotFound
			resp.StatusMsg = "address not found"
			return resp, nil
		}
		return nil, err
	}

	if addr.UserId != uint64(in.UserId) {
		resp.StatusCode = errno.AddressForbidden
		resp.StatusMsg = "address does not belong to user"
		return resp, nil
	}

	updated := false
	if detail := strings.TrimSpace(in.Address.Detail); detail != "" && detail != addr.Detail {
		addr.Detail = detail
		updated = true
	}

	setDefault := in.Address.IsDefault
	if setDefault && addr.IsDefault == 0 {
		addr.IsDefault = 1
		updated = true
	}

	if updated {
		if err := l.svcCtx.UserAddressModel.Update(l.ctx, addr); err != nil {
			return nil, err
		}
	}

	if setDefault {
		if err := ensureDefaultForUser(l.ctx, l.svcCtx.UserAddressModel, uint64(in.UserId), addr.Id); err != nil {
			return nil, err
		}
		addr.IsDefault = 1
	}

	refreshed, err := l.svcCtx.UserAddressModel.FindOne(l.ctx, addr.Id)
	if err != nil {
		return nil, err
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	resp.Address = addressToProto(refreshed)

	return resp, nil
}
