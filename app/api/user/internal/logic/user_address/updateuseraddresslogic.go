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

type UpdateUserAddressLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateUserAddressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserAddressLogic {
	return &UpdateUserAddressLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserAddressLogic) UpdateUserAddress(req *types.UpdateAddressRequest) (resp *types.Address, err error) {
	userId, err := util.UserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}

	in := &userservice.UpdateAddressRequest{
		UserId:    userId,
		AddressId: req.AddressId,
		Address: &userservice.AddressInput{
			Detail: req.Detail,
		},
	}

	res, err := l.svcCtx.UserRpc.UpdateAddress(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: update address failed: ", err)
		return nil, err
	}

	if res == nil {
		return nil, errors.New(int(errno.InternalError), "empty update address response")
	}

	if res.StatusCode != errno.StatusOK {
		l.Logger.Error("logic: update address failed: ", res)
		return nil, errors.New(int(res.StatusCode), res.StatusMsg)
	}

	addr := toAddress(res.Address)
	resp = &addr

	return
}
