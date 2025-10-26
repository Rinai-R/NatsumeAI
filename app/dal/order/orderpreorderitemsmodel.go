package order

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ OrderPreorderItemsModel = (*customOrderPreorderItemsModel)(nil)

type (
	// OrderPreorderItemsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customOrderPreorderItemsModel.
    OrderPreorderItemsModel interface {
        orderPreorderItemsModel
        // ListByPreorder returns preorder items for a preorder id
        ListByPreorder(ctx context.Context, preorderId int64) ([]*OrderPreorderItems, error)
    }

	customOrderPreorderItemsModel struct {
		*defaultOrderPreorderItemsModel
	}
)

// NewOrderPreorderItemsModel returns a model for the database table.
func NewOrderPreorderItemsModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) OrderPreorderItemsModel {
    return &customOrderPreorderItemsModel{
        defaultOrderPreorderItemsModel: newOrderPreorderItemsModel(conn, c, opts...),
    }
}

func (m *customOrderPreorderItemsModel) ListByPreorder(ctx context.Context, preorderId int64) ([]*OrderPreorderItems, error) {
    var rows []OrderPreorderItems
    query := fmt.Sprintf("select %s from %s where `preorder_id` = ? order by `id` asc", orderPreorderItemsRows, m.table)
    if err := m.QueryRowsNoCacheCtx(ctx, &rows, query, preorderId); err != nil {
        return nil, err
    }
    res := make([]*OrderPreorderItems, 0, len(rows))
    for i := range rows {
        res = append(res, &rows[i])
    }
    return res, nil
}