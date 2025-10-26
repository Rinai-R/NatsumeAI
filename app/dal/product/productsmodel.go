package product

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ProductsModel = (*customProductsModel)(nil)

type (
	// ProductsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customProductsModel.
	ProductsModel interface {
		productsModel
		FindAllProductId(ctx context.Context) ([]int64, error) 
	}

	customProductsModel struct {
		*defaultProductsModel
	}
)

// NewProductsModel returns a model for the database table.
func NewProductsModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) ProductsModel {
	return &customProductsModel{
		defaultProductsModel: newProductsModel(conn, c, opts...),
	}
}


func (m *customProductsModel) FindAllProductId(ctx context.Context) ([]int64, error) {
    query := fmt.Sprintf("SELECT `id` FROM %s", m.table)
    var ids []int64
    if err := m.QueryRowsNoCacheCtx(ctx, &ids, query); err != nil {
        return nil, err
    }
    return ids, nil
}
