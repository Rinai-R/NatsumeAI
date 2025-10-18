// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"NatsumeAI/app/api/user/internal/svc"
	"NatsumeAI/app/api/user/internal/types"
	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/services/user/userservice"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/x/errors"
)

type CreateUserAddressLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateUserAddressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateUserAddressLogic {
	return &CreateUserAddressLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateUserAddressLogic) CreateUserAddress(req *types.CreateAddressRequest) (resp *types.Address, err error) {
	userId, err := userIDFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}

	in := &userservice.CreateAddressRequest{
		UserId: userId,
		Address: &userservice.AddressInput{
			Detail:    req.Detail,
			IsDefault: req.IsDefault,
		},
	}

	res, err := l.svcCtx.UserRpc.CreateAddress(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: create address failed: ", err)
		return nil, err
	}
	if res == nil {
		return nil, errors.New(int(errno.InternalError), "empty create address response")
	}

	if res.StatusCode != errno.StatusOK {
		l.Logger.Error("logic: create address failed: ", res)
		return nil, errors.New(int(res.StatusCode), res.StatusMsg)
	}

	addr := toAddress(res.Address)
	resp = &addr
	return
}
