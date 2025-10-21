// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package inventory_manage

import (
	"net/http"

	"NatsumeAI/app/api/inventory/internal/logic/inventory_manage"
	"NatsumeAI/app/api/inventory/internal/svc"
	"NatsumeAI/app/api/inventory/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetInventoryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetInventoryRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := inventory_manage.NewGetInventoryLogic(r.Context(), svcCtx)
		resp, err := l.GetInventory(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
