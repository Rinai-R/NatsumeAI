package logic

import (
	"strings"
	"time"

	"NatsumeAI/app/common/consts/errno"
	couponmodel "NatsumeAI/app/dal/coupon"
	"NatsumeAI/app/services/coupon/coupon"
)

var (
	statusEnumToDB = map[coupon.CouponStatus]string{
		coupon.CouponStatus_COUPON_STATUS_UNUSED:  couponmodel.CouponStatusUnused,
		coupon.CouponStatus_COUPON_STATUS_LOCKED:  couponmodel.CouponStatusLocked,
		coupon.CouponStatus_COUPON_STATUS_USED:    couponmodel.CouponStatusUsed,
		coupon.CouponStatus_COUPON_STATUS_EXPIRED: couponmodel.CouponStatusExpired,
	}

	statusDBToEnum = map[string]coupon.CouponStatus{
		couponmodel.CouponStatusUnused:  coupon.CouponStatus_COUPON_STATUS_UNUSED,
		couponmodel.CouponStatusLocked:  coupon.CouponStatus_COUPON_STATUS_LOCKED,
		couponmodel.CouponStatusUsed:    coupon.CouponStatus_COUPON_STATUS_USED,
		couponmodel.CouponStatusExpired: coupon.CouponStatus_COUPON_STATUS_EXPIRED,
	}
)

type bizError struct {
	code int32
	msg  string
}

func newBizError(code int32, msg string) *bizError {
	return &bizError{code: code, msg: msg}
}

func (e *bizError) Error() string {
	return e.msg
}

func statusFromEnum(st coupon.CouponStatus) (string, bool) {
	val, ok := statusEnumToDB[st]
	return val, ok
}

func statusToEnum(status string, tplEnd time.Time) coupon.CouponStatus {
	status = strings.ToUpper(status)
	if status == couponmodel.CouponStatusUnused && time.Now().After(tplEnd) {
		return coupon.CouponStatus_COUPON_STATUS_EXPIRED
	}
	if enum, ok := statusDBToEnum[status]; ok {
		return enum
	}
	return coupon.CouponStatus_COUPON_STATUS_UNKNOWN
}

func couponTypeToProto(t int64) coupon.CouponType {
	switch t {
	case int64(coupon.CouponType_COUPON_TYPE_CASH):
		return coupon.CouponType_COUPON_TYPE_CASH
	case int64(coupon.CouponType_COUPON_TYPE_PERCENT):
		return coupon.CouponType_COUPON_TYPE_PERCENT
	default:
		return coupon.CouponType_COUPON_TYPE_UNKNOWN
	}
}

// 计算可以折扣的金额
func calcDiscountAmount(t coupon.CouponType, orderAmount, discountAmount, discountPercent, maxDiscount int64) int64 {
	switch t {
	case coupon.CouponType_COUPON_TYPE_CASH:
		if discountAmount < 0 {
			return 0
		}
		if discountAmount > orderAmount {
			return orderAmount
		}
		return discountAmount
	case coupon.CouponType_COUPON_TYPE_PERCENT:
		if discountPercent <= 0 {
			return 0
		}
		amount := orderAmount * discountPercent / 100
		if maxDiscount > 0 && amount > maxDiscount {
			amount = maxDiscount
		}
		if amount > orderAmount {
			return orderAmount
		}
		return amount
	default:
		return 0
	}
}

func normalizePagination(page, pageSize int32) (limit, offset int) {
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	if page <= 0 {
		page = 1
	}
	limit = int(pageSize)
	offset = int((page - 1) * pageSize)
	return
}

func errCodeToMsg(code int32, fallback string) string {
	switch code {
	case errno.InvalidParam:
		return "invalid request payload"
	case errno.CouponNotFound:
		return "coupon not found"
	case errno.CouponSoldOut:
		return "coupon sold out"
	case errno.CouponAlreadyClaimed:
		return "coupon already claimed"
	case errno.CouponExpired:
		return "coupon expired"
	case errno.CouponStatusInvalid:
		return "coupon status invalid"
	case errno.CouponOwnershipInvalid:
		return "coupon not owned by user"
	}
	if fallback != "" {
		return fallback
	}
	return "internal error"
}
