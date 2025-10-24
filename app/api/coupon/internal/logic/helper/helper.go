package helper

import (
    "NatsumeAI/app/api/coupon/internal/types"
    couponsvc "NatsumeAI/app/services/coupon/couponservice"
)

func ToCouponItem(in *couponsvc.CouponInfo) *types.CouponItem {
    if in == nil {
        return nil
    }
    return &types.CouponItem{
        CouponId:        in.CouponId,
        CouponType:      int32(in.CouponType),
        Status:          int32(in.Status),
        DiscountAmount:  in.DiscountAmount,
        DiscountPercent: int32(in.DiscountPercent),
        MinSpendAmount:  in.MinSpendAmount,
        UserId:          in.UserId,
        LockedPreorder:  in.LockedPreorder,
        StartAt:         in.StartAt,
        EndAt:           in.EndAt,
        Source:          in.Source,
        Remarks:         in.Remarks,
    }
}

