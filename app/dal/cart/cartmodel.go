package cart

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ CartModel = (*customCartModel)(nil)

type (
	// CartModel is an interface to be customized, add more methods here,
	// and implement the added methods in customCartModel.
	CartModel interface {
		cartModel
		FindOneByUserProduct(ctx context.Context, userId, productId int64) (*Cart, error)
		ListByUserId(ctx context.Context, userId int64) ([]*Cart, error)
	}

	customCartModel struct {
		*defaultCartModel
	}
)

// NewCartModel returns a model for the database table.
func NewCartModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) CartModel {
	return &customCartModel{
		defaultCartModel: newCartModel(conn, c, opts...),
	}
}

func (m *customCartModel) FindOneByUserProduct(ctx context.Context, userId, productId int64) (*Cart, error) {
	query := fmt.Sprintf("select %s from %s where `user_id` = ? and `product_id` = ? limit 1", cartRows, m.table)
	var resp Cart
	err := m.QueryRowNoCacheCtx(ctx, &resp, query, userId, productId)
	switch err {
	case nil:
		return &resp, nil
	case sqlc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (m *customCartModel) ListByUserId(ctx context.Context, userId int64) ([]*Cart, error) {
	query := fmt.Sprintf("select %s from %s where `user_id` = ? order by `id` desc", cartRows, m.table)
	var resp []*Cart
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, userId)
	switch err {
	case nil:
		return resp, nil
	case sqlc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}
