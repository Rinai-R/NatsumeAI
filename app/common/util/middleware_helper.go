package util

import (
	"NatsumeAI/app/common/consts/biz"
	"NatsumeAI/app/common/consts/errno"
	"context"
	"net/http"

	"github.com/zeromicro/x/errors"
)

func UserIdFromCtx(ctx context.Context) (int64, error) {
	if ctx == nil {
		return 0, errors.New(int(errno.TokenEmpty), "missing context")
	}

	switch val := ctx.Value(biz.USER_KEY).(type) {
	case int64:
		return val, nil
	}

	return 0, errors.New(int(errno.TokenEmpty), "unauthorized")
}


func InjectUserId2Ctx(r *http.Request, userId int64) {
	ctx := context.WithValue(r.Context(), biz.USER_KEY, userId)
	*r = *r.WithContext(ctx)
}