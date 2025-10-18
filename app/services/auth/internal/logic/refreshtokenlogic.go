package logic

import (
	"context"
	"errors"

	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/services/auth/auth"
	"NatsumeAI/app/services/auth/internal/svc"

	"github.com/golang-jwt/jwt/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type RefreshTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRefreshTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshTokenLogic {
	return &RefreshTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RefreshTokenLogic) RefreshToken(in *auth.RefreshTokenRequest) (*auth.RefreshTokenResponse, error) {
	resp := &auth.RefreshTokenResponse{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
	}

	if in == nil {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "request is nil"
		return resp, nil
	}

	if in.RefreshToken == "" {
		resp.StatusCode = errno.TokenEmpty
		resp.StatusMsg = "refresh token is empty"
		return resp, nil
	}

	claims, err := parseToken(in.RefreshToken, l.svcCtx.Config.RefreshSecret)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			resp.StatusCode = errno.RefreshTokenExpired
			resp.StatusMsg = "refresh token expired"
			return resp, nil
		}
		var ve *jwt.ValidationError
		if errors.As(err, &ve) && ve.Errors&jwt.ValidationErrorExpired != 0 {
			resp.StatusCode = errno.RefreshTokenExpired
			resp.StatusMsg = "refresh token expired"
			return resp, nil
		}

		l.Logger.Errorf("refresh token parse failed: %v", err)
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "refresh token invalid"
		return resp, nil
	}

	tokenPair, _, err := buildTokenPair(l.svcCtx.Config, claims.UserID, claims.Username)
	if err != nil {
		l.Logger.Errorf("refresh token generate pair failed: %v", err)
		return nil, err
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	resp.Token = tokenPair

	return resp, nil
}
