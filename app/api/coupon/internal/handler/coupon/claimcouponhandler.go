// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package coupon

import (
    "net/http"

    "NatsumeAI/app/api/coupon/internal/logic/coupon"
    "NatsumeAI/app/api/coupon/internal/svc"
    "NatsumeAI/app/api/coupon/internal/types"
    "github.com/zeromicro/go-zero/rest/httpx"
)

func ClaimCouponHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req types.ClaimCouponRequest
        if err := httpx.Parse(r, &req); err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
            return
        }

        l := coupon.NewClaimCouponLogic(r.Context(), svcCtx)
        resp, err := l.ClaimCoupon(&req)
        if err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
        } else {
            httpx.OkJsonCtx(r.Context(), w, resp)
        }
    }
}

