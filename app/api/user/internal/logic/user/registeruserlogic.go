// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"NatsumeAI/app/api/user/internal/logic/helper"
	"NatsumeAI/app/api/user/internal/svc"
	"NatsumeAI/app/api/user/internal/types"
	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/services/user/userservice"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/x/errors"
)

type RegisterUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterUserLogic {
	return &RegisterUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterUserLogic) RegisterUser(req *types.RegisterUserRequest) (resp *types.RegisterUserResponse, err error) {
	in := &userservice.RegisterUserRequest{
		Username: req.Username,
		Password: req.Password,
	}

	res, err := l.svcCtx.UserRpc.RegisterUser(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: register user failed: ", err)
		return nil, err
	}

	if res == nil {
		return nil, errors.New(int(errno.InternalError), "empty register response")
	}

	if res.StatusCode != errno.StatusOK {
		l.Logger.Error("logic: register user failed: ", res)
		return nil, errors.New(int(res.StatusCode), res.StatusMsg)
	}

	if res.User == nil {
		return nil, errors.New(int(errno.InternalError), "missing user profile")
	}

	resp = &types.RegisterUserResponse{
		User: helper.ToUserProfile(res.User),
	}

	return
}
