package logic

import (
	"context"
	"errors"
	"time"

	"NatsumeAI/app/common/consts/errno"
	couponmodel "NatsumeAI/app/dal/coupon"
	"NatsumeAI/app/services/coupon/coupon"
	"NatsumeAI/app/services/coupon/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ValidateCouponLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewValidateCouponLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ValidateCouponLogic {
	return &ValidateCouponLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 校验优惠券可用性
func (l *ValidateCouponLogic) ValidateCoupon(in *coupon.ValidateCouponReq) (*coupon.ValidateCouponResp, error) {
	resp := &coupon.ValidateCouponResp{
		StatusCode: errno.InternalError,
		StatusMsg:  errCodeToMsg(errno.InternalError, ""),
	}

	if in == nil || in.UserId <= 0 || in.CouponId <= 0 || in.OrderAmount <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = errCodeToMsg(errno.InvalidParam, "")
		return resp, nil
	}

	row, err := l.svcCtx.CouponInstancesModel.FindDetail(l.ctx, l.svcCtx.MysqlConn, in.CouponId, in.UserId)
	if err != nil {
		if errors.Is(err, couponmodel.ErrNotFound) {
			resp.StatusCode = errno.CouponNotFound
			resp.StatusMsg = errCodeToMsg(errno.CouponNotFound, "coupon not found")
			return resp, nil
		}
		return nil, err
	}

	now := time.Now()
	if now.Before(row.StartAt) || now.After(row.EndAt) {
		resp.StatusCode = errno.CouponExpired
		resp.StatusMsg = errCodeToMsg(errno.CouponExpired, "")
		resp.Valid = false
		return resp, nil
	}

	if row.Status != couponmodel.CouponStatusUnused {
		resp.StatusCode = errno.CouponStatusInvalid
		resp.StatusMsg = errCodeToMsg(errno.CouponStatusInvalid, "")
		resp.Valid = false
		return resp, nil
	}

	if in.OrderAmount < row.MinSpendAmount {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "order amount not eligible"
		resp.Valid = false
		return resp, nil
	}

	cType := couponTypeToProto(row.CouponType)
	discount := calcDiscountAmount(cType, in.OrderAmount, row.DiscountAmount, row.DiscountPercent, row.DiscountAmount)

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	resp.Valid = discount > 0
	resp.DiscountAmount = discount
	resp.CouponType = cType
	return resp, nil
}
