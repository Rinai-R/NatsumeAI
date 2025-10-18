package util

import (
	"net/http"
	"time"

	"NatsumeAI/app/common/consts/biz"
)

// 封装 setcookie
func SetTokenCookies(w http.ResponseWriter, accessToken string, accessExpiresIn int64, refreshToken string) {
	if accessToken != "" {
		ttl := biz.TokenExpire
		if accessExpiresIn > 0 {
			ttl = time.Duration(accessExpiresIn) * time.Second
		}
		expireAt := time.Now().Add(ttl)
		http.SetCookie(w, &http.Cookie{
			Name:     biz.ACCESSTOKEN,
			Value:    accessToken,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  expireAt,
			MaxAge:   int(ttl.Seconds()),
		})
	}

	if refreshToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     biz.REFRESHTOKEN,
			Value:    refreshToken,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(biz.TokenRenewalExpire),
			MaxAge:   int(biz.TokenRenewalExpire.Seconds()),
		})
	}
}
