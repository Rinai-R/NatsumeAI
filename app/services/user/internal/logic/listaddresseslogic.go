package logic

import (
	"context"

	"NatsumeAI/app/common/consts/errno"
	model "NatsumeAI/app/dal/user"
	"NatsumeAI/app/services/user/internal/svc"
	"NatsumeAI/app/services/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAddressesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAddressesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAddressesLogic {
	return &ListAddressesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListAddressesLogic) ListAddresses(in *user.ListAddressesRequest) (*user.ListAddressesResponse, error) {
	resp := &user.ListAddressesResponse{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
	}

	if in == nil {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "request is nil"
		return resp, nil
	}

	if in.UserId <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid user id"
		return resp, nil
	}

	if _, err := l.svcCtx.UserModel.FindOne(l.ctx, uint64(in.UserId)); err != nil {
		if err == model.ErrNotFound {
			resp.StatusCode = errno.UserNotFound
			resp.StatusMsg = "user not found"
			return resp, nil
		}
		return nil, err
	}

	addresses, err := l.svcCtx.UserAddressModel.FindByUserId(l.ctx, uint64(in.UserId))
	if err != nil {
		if err != model.ErrNotFound {
			return nil, err
		}
		addresses = nil
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	resp.Addresses = addressesToProto(addresses)

	return resp, nil
}
