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

type ClaimCouponLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewClaimCouponLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ClaimCouponLogic {
	return &ClaimCouponLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 领取优惠券
func (l *ClaimCouponLogic) ClaimCoupon(in *coupon.ClaimCouponReq) (*coupon.ClaimCouponResp, error) {
	resp := &coupon.ClaimCouponResp{
		StatusCode: int64(errno.InternalError),
		StatusMsg:  errCodeToMsg(errno.InternalError, ""),
	}

	if in == nil || in.UserId <= 0 || in.CouponId <= 0 {
		resp.StatusCode = int64(errno.InvalidParam)
		resp.StatusMsg = errCodeToMsg(errno.InvalidParam, "")
		return resp, nil
	}

	err := l.svcCtx.MysqlConn.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		tpl, err := l.svcCtx.CouponsModel.FindOneForUpdate(ctx, session, in.CouponId)
		if err != nil {
			if errors.Is(err, couponmodel.ErrNotFound) {
				return newBizError(errno.CouponNotFound, "coupon not found")
			}
			return err
		}

		now := time.Now()
		if now.Before(tpl.StartAt) {
			return newBizError(errno.CouponExpired, "coupon not started")
		}
		if now.After(tpl.EndAt) {
			return newBizError(errno.CouponExpired, "coupon expired")
		}

		if tpl.PerUserLimit > 0 {
			count, err := l.svcCtx.CouponInstancesModel.CountByUserCouponWithSession(ctx, session, in.UserId, in.CouponId)
			if err != nil {
				return err
			}
			if count >= tpl.PerUserLimit {
				return newBizError(errno.CouponAlreadyClaimed, "user reach coupon limit")
			}
		}

		if _, err := l.svcCtx.CouponInstancesModel.InsertWithSession(ctx, session, in.CouponId, in.UserId); err != nil {
			return err
		}

		if err := l.svcCtx.CouponsModel.IncrementIssuedWithSession(ctx, session, in.CouponId); err != nil {
			if errors.Is(err, couponmodel.ErrCouponSoldOut) {
				return newBizError(errno.CouponSoldOut, "coupon sold out")
			}
			return err
		}

		return nil
	})

	if err != nil {
		if be, ok := err.(*bizError); ok {
			resp.StatusCode = int64(be.code)
			resp.StatusMsg = errCodeToMsg(be.code, be.msg)
			return resp, nil
		}
		return nil, err
	}

	resp.StatusCode = int64(errno.StatusOK)
	resp.StatusMsg = "ok"
	return resp, nil
}
