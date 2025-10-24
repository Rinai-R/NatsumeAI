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

type RedeemCouponLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRedeemCouponLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RedeemCouponLogic {
	return &RedeemCouponLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 核销优惠券
func (l *RedeemCouponLogic) RedeemCoupon(in *coupon.RedeemCouponReq) (*coupon.RedeemCouponResp, error) {
	resp := &coupon.RedeemCouponResp{
		StatusCode: errno.InternalError,
		StatusMsg:  errCodeToMsg(errno.InternalError, ""),
	}

	if in == nil || in.UserId <= 0 || in.CouponId <= 0 || in.OrderId <= 0 || in.OrderAmount <= 0 {
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

		now := time.Now()
		if now.After(detail.EndAt) {
			return newBizError(errno.CouponExpired, "coupon expired")
		}

		if in.OrderAmount < detail.MinSpendAmount {
			return newBizError(errno.InvalidParam, "order amount not eligible")
		}

		cType := couponTypeToProto(detail.CouponType)
		discount := calcDiscountAmount(cType, in.OrderAmount, detail.DiscountAmount, detail.DiscountPercent, detail.DiscountAmount)
		if discount <= 0 {
			return newBizError(errno.CouponStatusInvalid, "coupon has no discount")
		}

		if err := l.svcCtx.CouponInstancesModel.RedeemWithSession(ctx, session, detail.InstanceId, detail.UserId, in.OrderId, now); err != nil {
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
