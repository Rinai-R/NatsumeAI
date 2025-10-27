package order

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ OrderPreordersModel = (*customOrderPreordersModel)(nil)

type (
    // OrderPreordersModel is an interface to be customized, add more methods here,
    // and implement the added methods in customOrderPreordersModel.
    OrderPreordersModel interface {
        orderPreordersModel
        // CancelIfPendingAndNoOrder sets status to CANCELLED only if it's still PENDING and no order exists.
        // Returns true if a row was updated.
        CancelIfPendingAndNoOrder(ctx context.Context, preorderId int64) (bool, error)
        // InsertWithId inserts a preorder row with a specific preorder_id (non-auto-increment primary key).
        InsertWithId(ctx context.Context, data *OrderPreorders) (sql.Result, error)
        // PlaceIfPending sets preorder status to PLACED if it's still PENDING.
        // Returns true if the row was updated.
        PlaceIfPending(ctx context.Context, preorderId int64) (bool, error)
        // PlaceIfPendingWithSession same as PlaceIfPending but within a given session.
        PlaceIfPendingWithSession(ctx context.Context, session sqlx.Session, preorderId int64) (bool, error)
        // MarkReadyIfPending promotes status from PENDING to READY atomically.
        MarkReadyIfPending(ctx context.Context, preorderId int64) (bool, error)
        // PlaceIfReady/WithSession sets status to PLACED only when current status is READY and not expired.
        PlaceIfReady(ctx context.Context, preorderId int64) (bool, error)
        PlaceIfReadyWithSession(ctx context.Context, session sqlx.Session, preorderId int64) (bool, error)
    }

    customOrderPreordersModel struct {
        *defaultOrderPreordersModel
    }
)

// NewOrderPreordersModel returns a model for the database table.
func NewOrderPreordersModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) OrderPreordersModel {
    return &customOrderPreordersModel{
        defaultOrderPreordersModel: newOrderPreordersModel(conn, c, opts...),
    }
}

// Override core CRUD to use no-cache paths for consistency.
func (m *customOrderPreordersModel) FindOne(ctx context.Context, preorderId int64) (*OrderPreorders, error) {
    var resp OrderPreorders
    query := fmt.Sprintf("select %s from %s where `preorder_id` = ? limit 1", orderPreordersRows, m.table)
    if err := m.QueryRowNoCacheCtx(ctx, &resp, query, preorderId); err != nil {
        return nil, err
    }
    return &resp, nil
}

func (m *customOrderPreordersModel) Insert(ctx context.Context, data *OrderPreorders) (sql.Result, error) {
    query := fmt.Sprintf("insert into %s (%s) values (?, ?, ?, ?, ?, ?)", m.table, orderPreordersRowsExpectAutoSet)
    return m.ExecNoCacheCtx(ctx, query, data.UserId, data.CouponId, data.OriginalAmount, data.FinalAmount, data.Status, data.ExpireAt)
}

func (m *customOrderPreordersModel) Update(ctx context.Context, data *OrderPreorders) error {
    query := fmt.Sprintf("update %s set %s where `preorder_id` = ?", m.table, orderPreordersRowsWithPlaceHolder)
    _, err := m.ExecNoCacheCtx(ctx, query, data.UserId, data.CouponId, data.OriginalAmount, data.FinalAmount, data.Status, data.ExpireAt, data.PreorderId)
    return err
}

func (m *customOrderPreordersModel) Delete(ctx context.Context, preorderId int64) error {
    query := fmt.Sprintf("delete from %s where `preorder_id` = ?", m.table)
    _, err := m.ExecNoCacheCtx(ctx, query, preorderId)
    return err
}

func (m *customOrderPreordersModel) CancelIfPendingAndNoOrder(ctx context.Context, preorderId int64) (bool, error) {
    orderPreordersPreorderIdKey := fmt.Sprintf("%s%v", cacheOrderPreordersPreorderIdPrefix, preorderId)
    res, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
        // 当订单不存在且预订单状态仍为 PENDING 或 READY 时，取消预订单。
        query := fmt.Sprintf("update %s op set `status` = ? where op.`preorder_id` = ? and op.`status` in (?, ?) and not exists (select 1 from `orders` o where o.`preorder_id` = op.`preorder_id` limit 1)", m.table)
        return conn.ExecCtx(ctx, query, "CANCELLED", preorderId, "PENDING", "READY")
    }, orderPreordersPreorderIdKey)
    if err != nil {
        return false, err
    }
    n, _ := res.RowsAffected()
    return n > 0, nil
}

func (m *customOrderPreordersModel) InsertWithId(ctx context.Context, data *OrderPreorders) (sql.Result, error) {
    orderPreordersPreorderIdKey := fmt.Sprintf("%s%v", cacheOrderPreordersPreorderIdPrefix, data.PreorderId)
    ret, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
        // explicit insert including preorder_id for snowflake-style IDs
        query := fmt.Sprintf("insert into %s (`preorder_id`,`user_id`,`coupon_id`,`original_amount`,`final_amount`,`status`,`expire_at`) values (?, ?, ?, ?, ?, ?, ?)", m.table)
        return conn.ExecCtx(ctx, query, data.PreorderId, data.UserId, data.CouponId, data.OriginalAmount, data.FinalAmount, data.Status, data.ExpireAt)
    }, orderPreordersPreorderIdKey)
    return ret, err
}

func (m *customOrderPreordersModel) PlaceIfPending(ctx context.Context, preorderId int64) (bool, error) {
    orderPreordersPreorderIdKey := fmt.Sprintf("%s%v", cacheOrderPreordersPreorderIdPrefix, preorderId)
    res, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
        query := fmt.Sprintf("update %s set `status` = ? where `preorder_id` = ? and `status` = ? and `expire_at` > now()", m.table)
        return conn.ExecCtx(ctx, query, "PLACED", preorderId, "PENDING")
    }, orderPreordersPreorderIdKey)
    if err != nil {
        return false, err
    }
    n, _ := res.RowsAffected()
    return n > 0, nil
}

func (m *customOrderPreordersModel) PlaceIfPendingWithSession(ctx context.Context, session sqlx.Session, preorderId int64) (bool, error) {
    query := fmt.Sprintf("update %s set `status` = ? where `preorder_id` = ? and `status` = ? and `expire_at` > now()", m.table)
    res, err := session.ExecCtx(ctx, query, "PLACED", preorderId, "PENDING")
    if err != nil {
        return false, err
    }
    n, _ := res.RowsAffected()
    return n > 0, nil
}

func (m *customOrderPreordersModel) MarkReadyIfPending(ctx context.Context, preorderId int64) (bool, error) {
    orderPreordersPreorderIdKey := fmt.Sprintf("%s%v", cacheOrderPreordersPreorderIdPrefix, preorderId)
    res, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
        query := fmt.Sprintf("update %s set `status` = ? where `preorder_id` = ? and `status` = ?", m.table)
        return conn.ExecCtx(ctx, query, "READY", preorderId, "PENDING")
    }, orderPreordersPreorderIdKey)
    if err != nil {
        return false, err
    }
    n, _ := res.RowsAffected()
    return n > 0, nil
}

func (m *customOrderPreordersModel) PlaceIfReady(ctx context.Context, preorderId int64) (bool, error) {
    orderPreordersPreorderIdKey := fmt.Sprintf("%s%v", cacheOrderPreordersPreorderIdPrefix, preorderId)
    res, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
        query := fmt.Sprintf("update %s set `status` = ? where `preorder_id` = ? and `status` = ? and `expire_at` > now()", m.table)
        return conn.ExecCtx(ctx, query, "PLACED", preorderId, "READY")
    }, orderPreordersPreorderIdKey)
    if err != nil {
        return false, err
    }
    n, _ := res.RowsAffected()
    return n > 0, nil
}

func (m *customOrderPreordersModel) PlaceIfReadyWithSession(ctx context.Context, session sqlx.Session, preorderId int64) (bool, error) {
    query := fmt.Sprintf("update %s set `status` = ? where `preorder_id` = ? and `status` = ? and `expire_at` > now()", m.table)
    res, err := session.ExecCtx(ctx, query, "PLACED", preorderId, "READY")
    if err != nil {
        return false, err
    }
    n, _ := res.RowsAffected()
    return n > 0, nil
}
