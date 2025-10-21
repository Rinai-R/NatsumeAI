// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"NatsumeAI/app/api/user/internal/svc"
	"NatsumeAI/app/api/user/internal/types"
	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/common/util"
	"NatsumeAI/app/services/user/userservice"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/x/errors"
)

type ListUserAddressesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListUserAddressesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListUserAddressesLogic {
	return &ListUserAddressesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListUserAddressesLogic) ListUserAddresses(req *types.ListAddressesRequest) (resp *types.ListAddressesResponse, err error) {
	userId, err := util.UserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}

	in := &userservice.ListAddressesRequest{
		UserId: userId,
	}

	res, err := l.svcCtx.UserRpc.ListAddresses(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: list addresses failed: ", err)
		return nil, err
	}

	if res == nil {
		return nil, errors.New(int(errno.InternalError), "empty list addresses response")
	}

	if res.StatusCode != errno.StatusOK {
		l.Logger.Error("logic: list addresses failed: ", res)
		return nil, errors.New(int(res.StatusCode), res.StatusMsg)
	}

	resp = &types.ListAddressesResponse{
		Addresses: toAddressSlice(res.Addresses),
	}

	return
}
