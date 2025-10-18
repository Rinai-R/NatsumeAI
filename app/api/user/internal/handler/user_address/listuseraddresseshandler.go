// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	logic "NatsumeAI/app/api/user/internal/logic/user_address"
	"NatsumeAI/app/api/user/internal/svc"
	"NatsumeAI/app/api/user/internal/types"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListUserAddressesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListAddressesRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewListUserAddressesLogic(r.Context(), svcCtx)
		resp, err := l.ListUserAddresses(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
