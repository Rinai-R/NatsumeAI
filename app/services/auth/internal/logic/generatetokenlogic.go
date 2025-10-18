package logic

import (
	"context"
	"strings"

	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/services/auth/auth"
	"NatsumeAI/app/services/auth/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GenerateTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGenerateTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GenerateTokenLogic {
	return &GenerateTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GenerateTokenLogic) GenerateToken(in *auth.GenerateTokenRequest) (*auth.GenerateTokenResponse, error) {
	resp := &auth.GenerateTokenResponse{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
	}

	if in == nil {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "request is nil"
		return resp, nil
	}

	username := strings.TrimSpace(in.Username)
	if in.UserId <= 0 || username == "" {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "user id or username is invalid"
		return resp, nil
	}

	tokenPair, _, err := buildTokenPair(l.svcCtx.Config, in.UserId, username)
	if err != nil {
		l.Logger.Errorf("generate token failed: %v", err)
		return nil, err
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	resp.Token = tokenPair

	return resp, nil
}
