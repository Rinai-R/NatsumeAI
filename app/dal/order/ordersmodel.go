package order

import (
    "context"
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
        // ListByUser returns orders for a user with pagination (desc by order_id)
        ListByUser(ctx context.Context, userId int64, offset, limit int64) ([]*Orders, error)
        // CountByUser returns total orders for a user
        CountByUser(ctx context.Context, userId int64) (int64, error)
    }

	customOrdersModel struct {
		*defaultOrdersModel
	}
)

// NewOrdersModel returns a model for the database table.
func NewOrdersModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) OrdersModel {
    return &customOrdersModel{
        defaultOrdersModel: newOrdersModel(conn, c, opts...),
    }
}

func (m *customOrdersModel) ListByUser(ctx context.Context, userId int64, offset, limit int64) ([]*Orders, error) {
    if limit <= 0 {
        limit = 10
    }
    if offset < 0 {
        offset = 0
    }
    var rows []Orders
    query := fmt.Sprintf("select %s from %s where `user_id` = ? order by `order_id` desc limit ? offset ?", ordersRows, m.table)
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
    q := fmt.Sprintf("select count(1) from %s where `user_id` = ?", m.table)
    if err := m.QueryRowNoCacheCtx(ctx, &total, q, userId); err != nil {
        return 0, err
    }
    return total, nil
}
