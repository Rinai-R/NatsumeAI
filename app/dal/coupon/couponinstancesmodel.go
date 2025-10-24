package coupon

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ CouponInstancesModel = (*customCouponInstancesModel)(nil)

type (
	// CouponInstancesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customCouponInstancesModel.
	CouponInstancesModel interface {
		couponInstancesModel
		InsertWithSession(ctx context.Context, session sqlx.Session, couponId int64, userId int64) (int64, error)
		CountByUserCouponWithSession(ctx context.Context, session sqlx.Session, userId, couponId int64) (int64, error)
		FindDetailForUpdate(ctx context.Context, session sqlx.Session, instanceId, userId int64) (*CouponInstanceDetail, error)
		FindDetail(ctx context.Context, conn sqlx.SqlConn, instanceId, userId int64) (*CouponInstanceDetail, error)
		LockWithSession(ctx context.Context, session sqlx.Session, instanceId, userId, orderId int64, lockTime time.Time) error
		ReleaseWithSession(ctx context.Context, session sqlx.Session, instanceId, userId, orderId int64) error
		RedeemWithSession(ctx context.Context, session sqlx.Session, instanceId, userId, orderId int64, usedAt time.Time) error
		ListUserCoupons(ctx context.Context, conn sqlx.SqlConn, userId int64, status string, offset, limit int) ([]*CouponInstanceDetail, int64, error)
	}

	customCouponInstancesModel struct {
		*defaultCouponInstancesModel
	}
)

// NewCouponInstancesModel returns a model for the database table.
func NewCouponInstancesModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) CouponInstancesModel {
	return &customCouponInstancesModel{
		defaultCouponInstancesModel: newCouponInstancesModel(conn, c, opts...),
	}
}

type CouponInstanceDetail struct {
	InstanceId      int64        `db:"instance_id"`
	CouponId        int64        `db:"coupon_id"`
	UserId          int64        `db:"user_id"`
	Status          string       `db:"status"`
	LockedPreorder  int64        `db:"locked_preorder"`
	LockedAt        sql.NullTime `db:"locked_at"`
	UsedOrderId     int64        `db:"used_order_id"`
	UsedAt          sql.NullTime `db:"used_at"`
	CreatedAt       time.Time    `db:"created_at"`
	UpdatedAt       time.Time    `db:"updated_at"`
	CouponType      int64        `db:"coupon_type"`
	DiscountAmount  int64        `db:"discount_amount"`
	DiscountPercent int64        `db:"discount_percent"`
	MinSpendAmount  int64        `db:"min_spend_amount"`
	StartAt         time.Time    `db:"start_at"`
	EndAt           time.Time    `db:"end_at"`
	Source          string       `db:"source"`
	Remarks         string       `db:"remarks"`
}

func (m *customCouponInstancesModel) InsertWithSession(ctx context.Context, session sqlx.Session, couponId int64, userId int64) (int64, error) {
	query := fmt.Sprintf("insert into %s (`coupon_id`, `user_id`, `status`) values (?, ?, ?)", m.table)
	res, err := session.ExecCtx(ctx, query, couponId, userId, CouponStatusUnused)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (m *customCouponInstancesModel) CountByUserCouponWithSession(ctx context.Context, session sqlx.Session, userId, couponId int64) (int64, error) {
	query := fmt.Sprintf("select count(1) from %s where `coupon_id` = ? and `user_id` = ?", m.table)
	var cnt int64
	if err := session.QueryRowCtx(ctx, &cnt, query, couponId, userId); err != nil {
		return 0, err
	}
	return cnt, nil
}

func (m *customCouponInstancesModel) FindDetailForUpdate(ctx context.Context, session sqlx.Session, instanceId, userId int64) (*CouponInstanceDetail, error) {
	couponsTable := "`coupons`"
	query := fmt.Sprintf(`
SELECT
    ci.id          AS instance_id,
    ci.coupon_id   AS coupon_id,
    ci.user_id     AS user_id,
    ci.status      AS status,
    ci.locked_preorder,
    ci.locked_at,
    ci.used_order_id,
    ci.used_at,
    ci.created_at,
    ci.updated_at,
    c.coupon_type,
    c.discount_amount,
    c.discount_percent,
    c.min_spend_amount,
    c.start_at,
    c.end_at,
    c.source,
    c.remarks
FROM %s ci
JOIN %s c ON ci.coupon_id = c.id
WHERE ci.id = ? AND ci.user_id = ?
FOR UPDATE`, m.table, couponsTable)

	var detail CouponInstanceDetail
	if err := session.QueryRowCtx(ctx, &detail, query, instanceId, userId); err != nil {
		if err == sql.ErrNoRows || err == sqlx.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &detail, nil
}

func (m *customCouponInstancesModel) FindDetail(ctx context.Context, conn sqlx.SqlConn, instanceId, userId int64) (*CouponInstanceDetail, error) {
	couponsTable := "`coupons`"
	query := fmt.Sprintf(`
SELECT
    ci.id          AS instance_id,
    ci.coupon_id   AS coupon_id,
    ci.user_id     AS user_id,
    ci.status      AS status,
    ci.locked_preorder,
    ci.locked_at,
    ci.used_order_id,
    ci.used_at,
    ci.created_at,
    ci.updated_at,
    c.coupon_type,
    c.discount_amount,
    c.discount_percent,
    c.min_spend_amount,
    c.start_at,
    c.end_at,
    c.source,
    c.remarks
FROM %s ci
JOIN %s c ON ci.coupon_id = c.id
WHERE ci.id = ? AND ci.user_id = ?`, m.table, couponsTable)

	var detail CouponInstanceDetail
	if err := conn.QueryRowCtx(ctx, &detail, query, instanceId, userId); err != nil {
		if err == sql.ErrNoRows || err == sqlx.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &detail, nil
}

func (m *customCouponInstancesModel) LockWithSession(ctx context.Context, session sqlx.Session, instanceId, userId, orderId int64, lockTime time.Time) error {
	query := fmt.Sprintf("update %s set `status` = ?, `locked_preorder` = ?, `locked_at` = ? where `id` = ? and `user_id` = ? and `status` = ?", m.table)
	res, err := session.ExecCtx(ctx, query, CouponStatusLocked, orderId, lockTime, instanceId, userId, CouponStatusUnused)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrCouponStatusConflict
	}
	return nil
}

func (m *customCouponInstancesModel) ReleaseWithSession(ctx context.Context, session sqlx.Session, instanceId, userId, orderId int64) error {
	query := fmt.Sprintf("update %s set `status` = ?, `locked_preorder` = 0, `locked_at` = NULL where `id` = ? and `user_id` = ? and `status` = ? and `locked_preorder` = ?", m.table)
	res, err := session.ExecCtx(ctx, query, CouponStatusUnused, instanceId, userId, CouponStatusLocked, orderId)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrCouponStatusConflict
	}
	return nil
}

func (m *customCouponInstancesModel) RedeemWithSession(ctx context.Context, session sqlx.Session, instanceId, userId, orderId int64, usedAt time.Time) error {
	query := fmt.Sprintf("update %s set `status` = ?, `used_order_id` = ?, `used_at` = ?, `locked_preorder` = 0 where `id` = ? and `user_id` = ? and `status` = ? and `locked_preorder` = ?", m.table)
	res, err := session.ExecCtx(ctx, query, CouponStatusUsed, orderId, usedAt, instanceId, userId, CouponStatusLocked, orderId)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrCouponStatusConflict
	}
	return nil
}

func (m *customCouponInstancesModel) ListUserCoupons(ctx context.Context, conn sqlx.SqlConn, userId int64, status string, offset, limit int) ([]*CouponInstanceDetail, int64, error) {
	couponsTable := "`coupons`"
	query := fmt.Sprintf(`
SELECT
    ci.id          AS instance_id,
    ci.coupon_id   AS coupon_id,
    ci.user_id     AS user_id,
    ci.status      AS status,
    ci.locked_preorder,
    ci.locked_at,
    ci.used_order_id,
    ci.used_at,
    ci.created_at,
    ci.updated_at,
    c.coupon_type,
    c.discount_amount,
    c.discount_percent,
    c.min_spend_amount,
    c.start_at,
    c.end_at,
    c.source,
    c.remarks
FROM %s ci
JOIN %s c ON ci.coupon_id = c.id
WHERE ci.user_id = ?`, m.table, couponsTable)

	args := []any{userId}
	if status != "" {
		query += " AND ci.status = ?"
		args = append(args, status)
	}
	query += " ORDER BY ci.id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	var list []*CouponInstanceDetail
	if err := conn.QueryRowsCtx(ctx, &list, query, args...); err != nil {
		if err == sql.ErrNoRows || err == sqlx.ErrNotFound {
			return []*CouponInstanceDetail{}, 0, nil
		}
		return nil, 0, err
	}

	countQuery := fmt.Sprintf("select count(1) from %s where `user_id` = ?", m.table)
	countArgs := []any{userId}
	if status != "" {
		countQuery += " and `status` = ?"
		countArgs = append(countArgs, status)
	}

	var total int64
	if err := conn.QueryRowCtx(ctx, &total, countQuery, countArgs...); err != nil {
		return nil, 0, err
	}

	return list, total, nil
}
