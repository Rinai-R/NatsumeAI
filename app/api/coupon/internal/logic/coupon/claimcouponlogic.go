// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package coupon

import (
    "context"

    "NatsumeAI/app/api/coupon/internal/svc"
    "NatsumeAI/app/api/coupon/internal/types"
    "NatsumeAI/app/common/consts/errno"
    "NatsumeAI/app/common/util"
    couponsvc "NatsumeAI/app/services/coupon/couponservice"

    "github.com/zeromicro/go-zero/core/logx"
    "github.com/zeromicro/x/errors"
)

type ClaimCouponLogic struct {
    logx.Logger
    ctx    context.Context
    svcCtx *svc.ServiceContext
}

func NewClaimCouponLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ClaimCouponLogic {
    return &ClaimCouponLogic{
        Logger: logx.WithContext(ctx),
        ctx:    ctx,
        svcCtx: svcCtx,
    }
}

func (l *ClaimCouponLogic) ClaimCoupon(req *types.ClaimCouponRequest) (resp *types.CouponActionResponse, err error) {
    if req == nil || req.CampaignId <= 0 {
        return nil, errors.New(int(errno.InvalidParam), "invalid campaign id")
    }

    userID, err := util.UserIdFromCtx(l.ctx)
    if err != nil {
        return nil, err
    }

    in := &couponsvc.ClaimCouponReq{
        UserId:   userID,
        CouponId: req.CampaignId,
    }

    res, err := l.svcCtx.CouponRpc.ClaimCoupon(l.ctx, in)
    if err != nil {
        l.Logger.Error("logic: claim coupon rpc failed: ", err)
        return nil, err
    }
    if res == nil {
        return nil, errors.New(int(errno.InternalError), "empty claim response")
    }

    return &types.CouponActionResponse{
        StatusCode: int32(res.StatusCode),
        StatusMsg:  res.StatusMsg,
    }, nil
}

