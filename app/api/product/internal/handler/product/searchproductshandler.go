// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package product

import (
	"net/http"

	"NatsumeAI/app/api/product/internal/logic/product"
	"NatsumeAI/app/api/product/internal/svc"
	"NatsumeAI/app/api/product/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func SearchProductsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SearchProductsRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := product.NewSearchProductsLogic(r.Context(), svcCtx)
		resp, err := l.SearchProducts(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
