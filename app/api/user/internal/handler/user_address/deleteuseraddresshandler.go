// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"net/http"

	logic "NatsumeAI/app/api/user/internal/logic/user_address"
	"NatsumeAI/app/api/user/internal/svc"
	"NatsumeAI/app/api/user/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func DeleteUserAddressHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DeleteAddressRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewDeleteUserAddressLogic(r.Context(), svcCtx)
		err := l.DeleteUserAddress(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.Ok(w)
		}
	}
}
