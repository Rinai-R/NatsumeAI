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
        // InsertWithId inserts a preorder with a specified preorder_id
        InsertWithId(ctx context.Context, data *OrderPreorders) (sql.Result, error)
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

// InsertWithId inserts a preorder row specifying preorder_id explicitly.
func (m *customOrderPreordersModel) InsertWithId(ctx context.Context, data *OrderPreorders) (sql.Result, error) {
    orderPreordersPreorderIdKey := fmt.Sprintf("%s%v", cacheOrderPreordersPreorderIdPrefix, data.PreorderId)
    ret, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
        query := fmt.Sprintf("insert into %s (`preorder_id`,`user_id`,`coupon_id`,`original_amount`,`final_amount`,`status`,`expire_at`) values (?, ?, ?, ?, ?, ?, ?)", m.table)
        return conn.ExecCtx(ctx, query, data.PreorderId, data.UserId, data.CouponId, data.OriginalAmount, data.FinalAmount, data.Status, data.ExpireAt)
    }, orderPreordersPreorderIdKey)
    return ret, err
}
