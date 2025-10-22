// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package cart

import (
	"net/http"

	"NatsumeAI/app/api/cart/internal/logic/cart"
	"NatsumeAI/app/api/cart/internal/svc"
	"NatsumeAI/app/api/cart/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetCartItemsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetCartItemsRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := cart.NewGetCartItemsLogic(r.Context(), svcCtx)
		resp, err := l.GetCartItems(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
