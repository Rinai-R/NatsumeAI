package coupon

import (
	"errors"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var (
	ErrNotFound             = sqlx.ErrNotFound
	ErrCouponSoldOut        = errors.New("coupon sold out")
	ErrCouponStatusConflict = errors.New("coupon status conflict")
	ErrCouponOwnership      = errors.New("coupon ownership invalid")
)

const (
	CouponStatusUnused  = "UNUSED"
	CouponStatusLocked  = "LOCKED"
	CouponStatusUsed    = "USED"
	CouponStatusExpired = "EXPIRED"
)
