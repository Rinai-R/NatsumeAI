package logic

import (
	"context"

	"NatsumeAI/app/common/consts/errno"
	model "NatsumeAI/app/dal/user"
	"NatsumeAI/app/services/user/internal/svc"
	"NatsumeAI/app/services/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteAddressLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteAddressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteAddressLogic {
	return &DeleteAddressLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteAddressLogic) DeleteAddress(in *user.DeleteAddressRequest) (*user.DeleteAddressResponse, error) {
	resp := &user.DeleteAddressResponse{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
	}

	if in == nil {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "request is nil"
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

	if err := l.svcCtx.UserAddressModel.Delete(l.ctx, addr.Id); err != nil {
		return nil, err
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	resp.Deleted = true

	return resp, nil
}
