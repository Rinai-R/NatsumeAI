// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import (
	"context"
	"net/http"

	"NatsumeAI/app/common/consts/biz"
	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/common/util"
	"NatsumeAI/app/services/auth/auth"
	"NatsumeAI/app/services/auth/authservice"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/x/errors"
)

type AuthMiddleware struct {
	AuthRpc auth.AuthServiceClient
}

func NewAuthMiddleware(auth auth.AuthServiceClient) *AuthMiddleware {
	return &AuthMiddleware{
		AuthRpc: auth,
	}
}

func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accessToken := ""
		if cookie, err := r.Cookie(biz.ACCESSTOKEN); err == nil {
			accessToken = cookie.Value
		} else if headerToken := r.Header.Get(biz.ACCESSTOKEN); headerToken != "" {
			accessToken = headerToken
		}
		refreshToken := ""
		if cookie, err := r.Cookie(biz.REFRESHTOKEN); err == nil {
			refreshToken = cookie.Value
		} else if headerToken := r.Header.Get(biz.REFRESHTOKEN); headerToken != "" {
			refreshToken = headerToken
		}

		if refreshToken == "" || accessToken == "" {
			err := errors.New(int(errno.TokenEmpty), "token is null")
			httpx.Error(w, err)
			return
		}
		in := &authservice.ValidateTokenRequest{
			AccessToken: accessToken,
		}
		res, err := m.AuthRpc.ValidateToken(r.Context(), in)
		if err != nil {
			httpx.Error(w, err)
			return
		}
		switch res.StatusCode {
		case errno.AccessTokenExpired:
			ref_in := &authservice.RefreshTokenRequest{
				RefreshToken: refreshToken,
			}
			ref_res, err := m.AuthRpc.RefreshToken(r.Context(), ref_in)
			if err != nil {
				httpx.Error(w, err)
				return
			}

			if ref_res.StatusCode != errno.StatusOK {
				httpx.Error(w, errors.New(int(ref_res.StatusCode), ref_res.StatusMsg))
				return
			}
			// 刷新成功的话就 setcookie 然后 next
			if token := ref_res.Token; token != nil {
				util.SetTokenCookies(w, token.AccessToken, token.ExpiresIn, token.RefreshToken)
				ctx := context.WithValue(r.Context(), biz.USER_KEY, res.UserId)
				*r = *r.WithContext(ctx)
				next(w, r)
				return
			}
			httpx.Error(w, errors.New(int(errno.RefreshTokenExpired), "token refresh failed"))

		case errno.StatusOK:
			ctx := context.WithValue(r.Context(), biz.USER_KEY, res.UserId)
			*r = *r.WithContext(ctx)
			next(w, r)
		default:
			httpx.Error(w, errors.New(int(res.StatusCode), res.StatusMsg))
		}
	}
}
