package order

import (
    "context"
    "database/sql"
    "fmt"

    "github.com/zeromicro/go-zero/core/stores/cache"
    "github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ OrdersModel = (*customOrdersModel)(nil)

type (
    // OrdersModel is an interface to be customized, add more methods here,
    // and implement the added methods in customOrdersModel.
    OrdersModel interface {
        ordersModel
        // InsertWithSession inserts an order row within given session.
        InsertWithSession(ctx context.Context, session sqlx.Session, data *Orders) (sql.Result, error)
        // ListByUser returns paginated orders for a user ordered by created_at desc
        ListByUser(ctx context.Context, userId int64, offset, limit int64) ([]*Orders, error)
        // CountByUser returns total orders count for a user
        CountByUser(ctx context.Context, userId int64) (int64, error)
    }

    customOrdersModel struct {
        *defaultOrdersModel
    }
)

// NewOrdersModel returns a model for the database table with custom methods.
func NewOrdersModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) OrdersModel {
    return &customOrdersModel{
        defaultOrdersModel: newOrdersModel(conn, c, opts...),
    }
}

func (m *customOrdersModel) InsertWithSession(ctx context.Context, session sqlx.Session, data *Orders) (sql.Result, error) {
    query := fmt.Sprintf("insert into %s (%s) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", m.table, ordersRowsExpectAutoSet)
    return session.ExecCtx(ctx, query, data.PreorderId, data.UserId, data.CouponId, data.Status, data.TotalAmount, data.PayableAmount, data.PaidAmount, data.PaymentMethod, data.PaymentAt, data.ExpireTime, data.CancelReason, data.AddressSnapshot)
}

func (m *customOrdersModel) ListByUser(ctx context.Context, userId int64, offset, limit int64) ([]*Orders, error) {
    var rows []Orders
    query := fmt.Sprintf("select %s from %s where `user_id` = ? order by `created_at` desc limit ? offset ?", ordersRows, m.table)
    if err := m.QueryRowsNoCacheCtx(ctx, &rows, query, userId, limit, offset); err != nil {
        return nil, err
    }
    res := make([]*Orders, 0, len(rows))
    for i := range rows {
        res = append(res, &rows[i])
    }
    return res, nil
}

func (m *customOrdersModel) CountByUser(ctx context.Context, userId int64) (int64, error) {
    var total int64
    query := fmt.Sprintf("select count(1) from %s where `user_id` = ?", m.table)
    if err := m.QueryRowNoCacheCtx(ctx, &total, query, userId); err != nil {
        return 0, err
    }
    return total, nil
}

// No-cache overrides for core CRUD
func (m *customOrdersModel) Insert(ctx context.Context, data *Orders) (sql.Result, error) {
    query := fmt.Sprintf("insert into %s (%s) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", m.table, ordersRowsExpectAutoSet)
    return m.ExecNoCacheCtx(ctx, query, data.PreorderId, data.UserId, data.CouponId, data.Status, data.TotalAmount, data.PayableAmount, data.PaidAmount, data.PaymentMethod, data.PaymentAt, data.ExpireTime, data.CancelReason, data.AddressSnapshot)
}

func (m *customOrdersModel) FindOne(ctx context.Context, orderId int64) (*Orders, error) {
    var resp Orders
    query := fmt.Sprintf("select %s from %s where `order_id` = ? limit 1", ordersRows, m.table)
    if err := m.QueryRowNoCacheCtx(ctx, &resp, query, orderId); err != nil {
        return nil, err
    }
    return &resp, nil
}

func (m *customOrdersModel) FindOneByPreorderId(ctx context.Context, preorderId int64) (*Orders, error) {
    var resp Orders
    query := fmt.Sprintf("select %s from %s where `preorder_id` = ? limit 1", ordersRows, m.table)
    if err := m.QueryRowNoCacheCtx(ctx, &resp, query, preorderId); err != nil {
        return nil, err
    }
    return &resp, nil
}

func (m *customOrdersModel) Update(ctx context.Context, data *Orders) error {
    query := fmt.Sprintf("update %s set %s where `order_id` = ?", m.table, ordersRowsWithPlaceHolder)
    _, err := m.ExecNoCacheCtx(ctx, query, data.PreorderId, data.UserId, data.CouponId, data.Status, data.TotalAmount, data.PayableAmount, data.PaidAmount, data.PaymentMethod, data.PaymentAt, data.ExpireTime, data.CancelReason, data.AddressSnapshot, data.OrderId)
    return err
}

func (m *customOrdersModel) Delete(ctx context.Context, orderId int64) error {
    query := fmt.Sprintf("delete from %s where `order_id` = ?", m.table)
    _, err := m.ExecNoCacheCtx(ctx, query, orderId)
    return err
}
