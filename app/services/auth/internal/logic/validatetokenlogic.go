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

type ValidateTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewValidateTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ValidateTokenLogic {
	return &ValidateTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ValidateTokenLogic) ValidateToken(in *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error) {
	resp := &auth.ValidateTokenResponse{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
		Valid:      false,
	}

	if in == nil {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "request is nil"
		return resp, nil
	}

	if in.AccessToken == "" {
		resp.StatusCode = errno.TokenEmpty
		resp.StatusMsg = "access token is empty"
		return resp, nil
	}

	claims, err := parseToken(in.AccessToken, l.svcCtx.Config.AccessSecret)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			resp.StatusCode = errno.AccessTokenExpired
			resp.StatusMsg = "access token expired"
			return resp, nil
		}
		var ve *jwt.ValidationError
		if errors.As(err, &ve) && ve.Errors&jwt.ValidationErrorExpired != 0 {
			resp.StatusCode = errno.AccessTokenExpired
			resp.StatusMsg = "access token expired"
			return resp, nil
		}

		l.Logger.Errorf("validate token parse failed: %v", err)
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "access token invalid"
		return resp, nil
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	resp.Valid = true
	resp.UserId = claims.UserID
	resp.Username = claims.Username
	if claims.ExpiresAt != nil {
		resp.ExpiresAt = claims.ExpiresAt.Time.Unix()
	}

	return resp, nil
}
