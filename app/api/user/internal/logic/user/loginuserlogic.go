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

type LoginUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginUserLogic {
	return &LoginUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginUserLogic) LoginUser(req *types.LoginUserRequest) (resp *types.LoginUserResponse, err error) {
	in := &userservice.LoginUserRequest{
		Username: req.Username,
		Password: req.Password,
	}

	res, err := l.svcCtx.UserRpc.LoginUser(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: login user failed: ", err)
		return nil, err
	}

	if res == nil {
		return nil, errors.New(int(errno.InternalError), "empty login response")
	}

	if res.StatusCode != errno.StatusOK {
		l.Logger.Error("logic: login user failed: ", res)
		return nil, errors.New(int(res.StatusCode), res.StatusMsg)
	}

	resp = &types.LoginUserResponse{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		ExpiresIn:    res.ExpiresIn,
		User:         helper.ToUserProfile(res.User),
	}
	return
}
