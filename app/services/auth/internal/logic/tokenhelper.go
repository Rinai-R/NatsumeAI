package logic

import (
	"errors"
	"time"

	"NatsumeAI/app/services/auth/auth"
	"NatsumeAI/app/services/auth/internal/config"

	"github.com/golang-jwt/jwt/v4"
)

type jwtClaims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func buildTokenPair(cfg config.Config, userID int64, username string) (*auth.TokenPair, time.Time, error) {
	accessToken, accessExpireAt, err := signToken(cfg.AccessSecret, cfg.AccessExpire, userID, username)
	if err != nil {
		return nil, time.Time{}, err
	}

	refreshToken, _, err := signToken(cfg.RefreshSecret, cfg.RefreshExpire, userID, username)
	if err != nil {
		return nil, time.Time{}, err
	}

	token := &auth.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(cfg.AccessExpire.Seconds()),
	}
	return token, accessExpireAt, nil
}

func signToken(secret string, ttl time.Duration, userID int64, username string) (string, time.Time, error) {
	if secret == "" {
		return "", time.Time{}, errors.New("token secret is empty")
	}
	if ttl <= 0 {
		return "", time.Time{}, errors.New("token ttl must be positive")
	}

	now := time.Now()
	expireAt := now.Add(ttl)
	claims := jwtClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireAt),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expireAt, nil
}

func parseToken(tokenStr, secret string) (*jwtClaims, error) {
	if tokenStr == "" {
		return nil, errors.New("token is empty")
	}
	if secret == "" {
		return nil, errors.New("token secret is empty")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
