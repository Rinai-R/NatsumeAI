// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package order

import (
	"net/http"

	"NatsumeAI/app/api/order/internal/logic/order"
	"NatsumeAI/app/api/order/internal/svc"
	"NatsumeAI/app/api/order/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func CheckoutHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CheckoutRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := order.NewCheckoutLogic(r.Context(), svcCtx)
		resp, err := l.Checkout(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
