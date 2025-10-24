package logic

import (
	"context"
	"time"

	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/services/coupon/coupon"
	"NatsumeAI/app/services/coupon/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListUserCouponsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListUserCouponsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListUserCouponsLogic {
	return &ListUserCouponsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询用户优惠券列表
func (l *ListUserCouponsLogic) ListUserCoupons(in *coupon.ListUserCouponsReq) (*coupon.ListUserCouponsResp, error) {
	resp := &coupon.ListUserCouponsResp{
		StatusCode: errno.InternalError,
		StatusMsg:  errCodeToMsg(errno.InternalError, ""),
	}

	if in == nil || in.UserId <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = errCodeToMsg(errno.InvalidParam, "")
		return resp, nil
	}

	statusFilter := ""
	if in.Status != coupon.CouponStatus_COUPON_STATUS_UNKNOWN {
		if val, ok := statusFromEnum(in.Status); ok {
			statusFilter = val
		} else {
			resp.StatusCode = errno.InvalidParam
			resp.StatusMsg = errCodeToMsg(errno.InvalidParam, "invalid status filter")
			return resp, nil
		}
	}

	limit, offset := normalizePagination(in.Page, in.PageSize)

	rows, total, err := l.svcCtx.CouponInstancesModel.ListUserCoupons(l.ctx, l.svcCtx.MysqlConn, in.UserId, statusFilter, offset, limit)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	result := make([]*coupon.CouponInfo, 0, len(rows))
	for _, row := range rows {
		status := statusToEnum(row.Status, row.EndAt)
		// 过期了，标记一下
		if status != coupon.CouponStatus_COUPON_STATUS_EXPIRED && now.After(row.EndAt) {
			status = coupon.CouponStatus_COUPON_STATUS_EXPIRED
		}

		info := &coupon.CouponInfo{
			CouponId:        row.InstanceId,
			CouponType:      couponTypeToProto(row.CouponType),
			Status:          status,
			DiscountAmount:  row.DiscountAmount,
			DiscountPercent: int32(row.DiscountPercent),
			MinSpendAmount:  row.MinSpendAmount,
			UserId:          row.UserId,
			LockedPreorder:  row.LockedPreorder,
			StartAt:         row.StartAt.Unix(),
			EndAt:           row.EndAt.Unix(),
			Source:          row.Source,
			Remarks:         row.Remarks,
		}
		result = append(result, info)
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	resp.Coupons = result
	resp.Total = total
	return resp, nil
}
