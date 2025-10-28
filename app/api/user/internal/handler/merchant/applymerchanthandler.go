// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package merchant

import (
	"net/http"

	"NatsumeAI/app/api/user/internal/logic/merchant"
	"NatsumeAI/app/api/user/internal/svc"
	"NatsumeAI/app/api/user/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ApplyMerchantHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ApplyMerchantRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := merchant.NewApplyMerchantLogic(r.Context(), svcCtx)
		resp, err := l.ApplyMerchant(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
