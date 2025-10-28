// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package merchant

import (
    "net/http"

    "NatsumeAI/app/api/user/internal/logic/merchant"
    "NatsumeAI/app/api/user/internal/svc"
    "github.com/zeromicro/go-zero/rest/httpx"
)

func GetMerchantApplicationStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // parse path param
        var path struct{ ApplicationId int64 `path:"applicationId"` }
        if err := httpx.Parse(r, &path); err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
            return
        }
        l := merchant.NewGetMerchantApplicationStatusLogic(r.Context(), svcCtx)
        resp, err := l.GetMerchantApplicationStatus(path.ApplicationId)
        if err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
        } else {
            httpx.OkJsonCtx(r.Context(), w, resp)
        }
    }
}
