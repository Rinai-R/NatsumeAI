// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package coupon

import (
    "context"

    "NatsumeAI/app/api/coupon/internal/logic/helper"
    "NatsumeAI/app/api/coupon/internal/svc"
    "NatsumeAI/app/api/coupon/internal/types"
    "NatsumeAI/app/common/consts/errno"
    "NatsumeAI/app/common/util"
    couponsvc "NatsumeAI/app/services/coupon/couponservice"
    proto "NatsumeAI/app/services/coupon/coupon"

    "github.com/zeromicro/go-zero/core/logx"
    "github.com/zeromicro/x/errors"
)

type ListUserCouponsLogic struct {
    logx.Logger
    ctx    context.Context
    svcCtx *svc.ServiceContext
}

func NewListUserCouponsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListUserCouponsLogic {
    return &ListUserCouponsLogic{
        Logger: logx.WithContext(ctx),
        ctx:    ctx,
        svcCtx: svcCtx,
    }
}

func (l *ListUserCouponsLogic) ListUserCoupons(req *types.ListUserCouponsRequest) (resp *types.ListUserCouponsResponse, err error) {
    userID, err := util.UserIdFromCtx(l.ctx)
    if err != nil {
        return nil, err
    }

    in := &couponsvc.ListUserCouponsReq{
        UserId:   userID,
        Status:   proto.CouponStatus(req.Status),
        Page:     req.Page,
        PageSize: req.PageSize,
    }

    res, err := l.svcCtx.CouponRpc.ListUserCoupons(l.ctx, in)
    if err != nil {
        l.Logger.Error("logic: list user coupons rpc failed: ", err)
        return nil, err
    }
    if res == nil {
        return nil, errors.New(int(errno.InternalError), "empty list coupons response")
    }

    items := make([]types.CouponItem, 0, len(res.Coupons))
    for _, c := range res.Coupons {
        if v := helper.ToCouponItem(c); v != nil {
            items = append(items, *v)
        }
    }

    return &types.ListUserCouponsResponse{
        StatusCode: res.StatusCode,
        StatusMsg:  res.StatusMsg,
        Coupons:    items,
        Total:      res.Total,
    }, nil
}
