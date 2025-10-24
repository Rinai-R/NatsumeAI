package logic

import (
	"context"
	"time"

	"NatsumeAI/app/common/consts/errno"
	couponmodel "NatsumeAI/app/dal/coupon"
	"NatsumeAI/app/services/coupon/coupon"
	"NatsumeAI/app/services/coupon/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PublishCouponLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPublishCouponLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishCouponLogic {
	return &PublishCouponLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 发布/批量发券
func (l *PublishCouponLogic) PublishCoupon(in *coupon.PublishCouponReq) (*coupon.PublishCouponResp, error) {
	resp := &coupon.PublishCouponResp{
		StatusCode: errno.InternalError,
		StatusMsg:  errCodeToMsg(errno.InternalError, ""),
	}

	if in == nil {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = errCodeToMsg(errno.InvalidParam, "")
		return resp, nil
	}

	if in.CouponType == coupon.CouponType_COUPON_TYPE_UNKNOWN {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid coupon type"
		return resp, nil
	}

	startAt := time.Unix(in.StartAt, 0).UTC()
	endAt := time.Unix(in.EndAt, 0).UTC()
	if !endAt.After(startAt) {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid time range"
		return resp, nil
	}

	data := &couponmodel.Coupons{
		CouponType:      int64(in.CouponType),
		DiscountAmount:  in.DiscountAmount,
		DiscountPercent: int64(in.DiscountPercent),
		MinSpendAmount:  in.MinSpendAmount,
		TotalQuantity:   in.TotalIssue,
		PerUserLimit:    in.PerUserLimit,
		IssuedQuantity:  0,
		StartAt:         startAt,
		EndAt:           endAt,
		Source:          in.Source,
		Remarks:         in.Remarks,
	}

	result, err := l.svcCtx.CouponsModel.Insert(l.ctx, data)
	if err != nil {
		return nil, err
	}

	campaignID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	resp.CampaignId = campaignID
	return resp, nil
}
