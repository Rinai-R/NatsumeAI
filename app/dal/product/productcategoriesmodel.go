package product

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ProductCategoriesModel = (*customProductCategoriesModel)(nil)

type (
	// ProductCategoriesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customProductCategoriesModel.
	ProductCategoriesModel interface {
		productCategoriesModel
		ListByProductId(ctx context.Context, productId int64) ([]*ProductCategories, error)
		DeleteByProductId(ctx context.Context, productId int64) error
	}

	customProductCategoriesModel struct {
		*defaultProductCategoriesModel
	}
)

// NewProductCategoriesModel returns a model for the database table.
func NewProductCategoriesModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) ProductCategoriesModel {
	return &customProductCategoriesModel{
		defaultProductCategoriesModel: newProductCategoriesModel(conn, c, opts...),
	}
}

func (m *customProductCategoriesModel) ListByProductId(ctx context.Context, productId int64) ([]*ProductCategories, error) {
	var resp []*ProductCategories
	query := fmt.Sprintf("select %s from %s where `product_id` = ?", productCategoriesRows, m.table)
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, productId)
	switch err {
	case nil:
		return resp, nil
	case sqlc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (m *customProductCategoriesModel) DeleteByProductId(ctx context.Context, productId int64) error {
	items, err := m.ListByProductId(ctx, productId)
	if err != nil && err != ErrNotFound {
		return err
	}

	for _, item := range items {
		if item == nil {
			continue
		}
		if err := m.Delete(ctx, item.Id); err != nil && err != ErrNotFound {
			return err
		}
	}
	return nil
}
