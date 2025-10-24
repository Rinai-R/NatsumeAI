// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package coupon_manage

import (
    "context"

    "NatsumeAI/app/api/coupon/internal/svc"
    "NatsumeAI/app/api/coupon/internal/types"
    "NatsumeAI/app/common/consts/errno"
    couponsvc "NatsumeAI/app/services/coupon/couponservice"
    proto "NatsumeAI/app/services/coupon/coupon"

    "github.com/zeromicro/go-zero/core/logx"
    "github.com/zeromicro/x/errors"
)

type PublishCouponLogic struct {
    logx.Logger
    ctx    context.Context
    svcCtx *svc.ServiceContext
}

func NewPublishCouponLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishCouponLogic {
    return &PublishCouponLogic{
        Logger: logx.WithContext(ctx),
        ctx:    ctx,
        svcCtx: svcCtx,
    }
}

func (l *PublishCouponLogic) PublishCoupon(req *types.PublishCouponRequest) (resp *types.PublishCouponResponse, err error) {
    if req == nil {
        return nil, errors.New(int(errno.InvalidParam), "invalid publish payload")
    }

    in := &couponsvc.PublishCouponReq{
        CouponType:      proto.CouponType(req.CouponType),
        DiscountAmount:  req.DiscountAmount,
        DiscountPercent: int32(req.DiscountPercent),
        MinSpendAmount:  req.MinSpendAmount,
        TotalIssue:      req.TotalIssue,
        PerUserLimit:    req.PerUserLimit,
        StartAt:         req.StartAt,
        EndAt:           req.EndAt,
        Source:          req.Source,
        Remarks:         req.Remarks,
    }

    res, err := l.svcCtx.CouponRpc.PublishCoupon(l.ctx, in)
    if err != nil {
        l.Logger.Error("logic: publish coupon rpc failed: ", err)
        return nil, err
    }
    if res == nil {
        return nil, errors.New(int(errno.InternalError), "empty publish response")
    }

    return &types.PublishCouponResponse{
        StatusCode: res.StatusCode,
        StatusMsg:  res.StatusMsg,
        CampaignId: res.CampaignId,
    }, nil
}
