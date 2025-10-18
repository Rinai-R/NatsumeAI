package logic

import (
	"context"
	"strings"

	"NatsumeAI/app/common/consts/errno"
	model "NatsumeAI/app/dal/user"
	"NatsumeAI/app/services/auth/auth"
	"NatsumeAI/app/services/user/internal/svc"
	"NatsumeAI/app/services/user/user"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
)

type LoginUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginUserLogic {
	return &LoginUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LoginUserLogic) LoginUser(in *user.LoginUserRequest) (*user.LoginUserResponse, error) {
	resp := &user.LoginUserResponse{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
	}

	if in == nil {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "request is nil"
		return resp, nil
	}

	username := strings.TrimSpace(in.Username)
	password := strings.TrimSpace(in.Password)
	if username == "" || password == "" {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "username and password are required"
		return resp, nil
	}


	// 布隆过滤器
	if l.svcCtx.Bloom != nil {
		exists, err := l.svcCtx.Bloom.Exists([]byte(username))
		if err != nil {
			l.Logger.Errorf("login bloom exists failed: %v", err)
		} else if !exists {
			resp.StatusCode = errno.UserNotFound
			resp.StatusMsg = "user not found"
			return resp, nil
		}
	}

	dbUser, err := l.svcCtx.UserModel.FindOneByUsername(l.ctx, username)
	if err != nil {
		if err == model.ErrNotFound {
			resp.StatusCode = errno.UserNotFound
			resp.StatusMsg = "user not found"
			return resp, nil
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(password)); err != nil {
		resp.StatusCode = errno.InvalidCredentials
		resp.StatusMsg = "invalid credentials"
		return resp, nil
	}

	tokenReq := &auth.GenerateTokenRequest{
		UserId:   int64(dbUser.Id),
		Username: dbUser.Username,
	}
	tokenResp, err := l.svcCtx.AuthRpc.GenerateToken(l.ctx, tokenReq)
	if err != nil {
		return nil, err
	}
	if tokenResp.StatusCode != errno.StatusOK {
		return &user.LoginUserResponse{
			StatusCode: tokenResp.StatusCode,
			StatusMsg:  tokenResp.StatusMsg,
		}, nil
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	resp.AccessToken = tokenResp.Token.GetAccessToken()
	resp.RefreshToken = tokenResp.Token.GetRefreshToken()
	resp.ExpiresIn = tokenResp.Token.GetExpiresIn()
	resp.User = userToProfile(dbUser)

	return resp, nil
}
