package logic

import (
	"context"
	"errors"

	"NatsumeAI/app/common/consts/errno"
	couponmodel "NatsumeAI/app/dal/coupon"
	"NatsumeAI/app/services/coupon/coupon"
	"NatsumeAI/app/services/coupon/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ReleaseCouponLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReleaseCouponLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReleaseCouponLogic {
	return &ReleaseCouponLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 释放优惠券
func (l *ReleaseCouponLogic) ReleaseCoupon(in *coupon.ReleaseCouponReq) (*coupon.ReleaseCouponResp, error) {
	resp := &coupon.ReleaseCouponResp{
		StatusCode: errno.InternalError,
		StatusMsg:  errCodeToMsg(errno.InternalError, ""),
	}

	if in == nil || in.UserId <= 0 || in.CouponId <= 0 || in.OrderId <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = errCodeToMsg(errno.InvalidParam, "")
		return resp, nil
	}

	err := l.svcCtx.MysqlConn.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		detail, err := l.svcCtx.CouponInstancesModel.FindDetailForUpdate(ctx, session, in.CouponId, in.UserId)
		if err != nil {
			if errors.Is(err, couponmodel.ErrNotFound) {
				return newBizError(errno.CouponNotFound, "coupon not found")
			}
			return err
		}

		if detail.Status != couponmodel.CouponStatusLocked {
			return newBizError(errno.CouponStatusInvalid, "coupon not locked")
		}

		if detail.LockedPreorder != in.OrderId {
			return newBizError(errno.CouponOwnershipInvalid, "coupon locked by another order")
		}

		if err := l.svcCtx.CouponInstancesModel.ReleaseWithSession(ctx, session, detail.InstanceId, detail.UserId, in.OrderId); err != nil {
			if errors.Is(err, couponmodel.ErrCouponStatusConflict) {
				return newBizError(errno.CouponStatusInvalid, "coupon status invalid")
			}
			return err
		}

		return nil
	})

	if err != nil {
		if be, ok := err.(*bizError); ok {
			resp.StatusCode = be.code
			resp.StatusMsg = errCodeToMsg(be.code, be.msg)
			return resp, nil
		}
		return nil, err
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	return resp, nil
}
