package order

import (
    "context"
    "fmt"

    "github.com/zeromicro/go-zero/core/stores/cache"
    "github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ OrderItemsModel = (*customOrderItemsModel)(nil)

type (
	// OrderItemsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customOrderItemsModel.
    OrderItemsModel interface {
        orderItemsModel
        // ListByOrder returns items for an order
        ListByOrder(ctx context.Context, orderId int64) ([]*OrderItems, error)
    }

	customOrderItemsModel struct {
		*defaultOrderItemsModel
	}
)

// NewOrderItemsModel returns a model for the database table.
func NewOrderItemsModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) OrderItemsModel {
    return &customOrderItemsModel{
        defaultOrderItemsModel: newOrderItemsModel(conn, c, opts...),
    }
}

func (m *customOrderItemsModel) ListByOrder(ctx context.Context, orderId int64) ([]*OrderItems, error) {
    var rows []OrderItems
    query := fmt.Sprintf("select %s from %s where `order_id` = ? order by `id` asc", orderItemsRows, m.table)
    if err := m.QueryRowsNoCacheCtx(ctx, &rows, query, orderId); err != nil {
        return nil, err
    }
    res := make([]*OrderItems, 0, len(rows))
    for i := range rows {
        res = append(res, &rows[i])
    }
    return res, nil
}
