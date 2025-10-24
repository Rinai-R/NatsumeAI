package coupon

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ CouponsModel = (*customCouponsModel)(nil)

type (
	// CouponsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customCouponsModel.
	CouponsModel interface {
		couponsModel
		FindOneForUpdate(ctx context.Context, session sqlx.Session, id int64) (*Coupons, error)
		IncrementIssuedWithSession(ctx context.Context, session sqlx.Session, id int64) error
	}

	customCouponsModel struct {
		*defaultCouponsModel
	}
)

// NewCouponsModel returns a model for the database table.
func NewCouponsModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) CouponsModel {
	return &customCouponsModel{
		defaultCouponsModel: newCouponsModel(conn, c, opts...),
	}
}

func (m *customCouponsModel) FindOneForUpdate(ctx context.Context, session sqlx.Session, id int64) (*Coupons, error) {
	query := fmt.Sprintf("select %s from %s where `id` = ? limit 1 for update", couponsRows, m.table)
	var resp Coupons
	if err := session.QueryRowCtx(ctx, &resp, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		if err == sqlx.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &resp, nil
}

func (m *customCouponsModel) IncrementIssuedWithSession(ctx context.Context, session sqlx.Session, id int64) error {
	query := fmt.Sprintf("update %s set `issued_quantity` = `issued_quantity` + 1 where `id` = ? and (`total_quantity` = 0 or `issued_quantity` < `total_quantity`)", m.table)
	res, err := session.ExecCtx(ctx, query, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrCouponSoldOut
	}
	return nil
}
