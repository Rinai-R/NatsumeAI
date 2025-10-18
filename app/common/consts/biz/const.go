package biz

import "time"


type CtxKey string

const (
	USER_KEY CtxKey = "user_id"

	TokenExpire        = time.Hour * 2
	TokenRenewalExpire = time.Hour * 24 * 7

	REFRESHTOKEN = "refresh_token"
	ACCESSTOKEN = "access_token"
)