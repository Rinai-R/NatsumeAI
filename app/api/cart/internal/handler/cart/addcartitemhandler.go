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

func AddCartItemHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AddCartItemRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := cart.NewAddCartItemLogic(r.Context(), svcCtx)
		resp, err := l.AddCartItem(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
