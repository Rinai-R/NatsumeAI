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
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type LockCouponLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLockCouponLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LockCouponLogic {
	return &LockCouponLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 锁定优惠券
func (l *LockCouponLogic) LockCoupon(in *coupon.LockCouponReq) (*coupon.LockCouponResp, error) {
	resp := &coupon.LockCouponResp{
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

		now := time.Now()
		if now.After(detail.EndAt) {
			return newBizError(errno.CouponExpired, "coupon expired")
		}

		// 幂等：若已被同一预订单锁定，则视为成功
		if detail.Status == couponmodel.CouponStatusLocked && detail.LockedPreorder == in.OrderId {
			return nil
		}
		if detail.Status != couponmodel.CouponStatusUnused {
			return newBizError(errno.CouponStatusInvalid, "coupon not unused")
		}

		if err := l.svcCtx.CouponInstancesModel.LockWithSession(ctx, session, detail.InstanceId, detail.UserId, in.OrderId, now); err != nil {
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
