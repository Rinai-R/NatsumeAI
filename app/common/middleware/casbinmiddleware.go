package middleware

import (
	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/common/util"
	"fmt"
	"net/http"
	"strconv"

	"github.com/casbin/casbin/v2"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/x/errors"
)


type CasbinMiddleware struct {
	Enforcer *casbin.DistributedEnforcer
}

func NewCasbinMiddleware(e *casbin.DistributedEnforcer) *CasbinMiddleware {
	return &CasbinMiddleware{e}
}

func (m *CasbinMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, err := util.UserIdFromCtx(r.Context())
		if err != nil {
			err := errors.New(int(errno.TokenEmpty), "token is null")
			httpx.Error(w, err)
			return
		}
        sub := strconv.FormatInt(userId, 10)
        ok, err := m.Enforcer.Enforce(sub, r.URL.Path, r.Method)
		fmt.Println(ok, err, sub, r.URL.Path, r.Method)
		if err != nil {
			err := errors.New(int(errno.CasbinError), "casbin error "+err.Error())
			httpx.Error(w, err)
			return
		}
		if !ok {
			err := errors.New(int(errno.NoPermission), "NoPermission")
			httpx.Error(w, err)
			return
		}
		next(w, r)
	}
}
