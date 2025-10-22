package inventory

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ InventoryModel = (*customInventoryModel)(nil)

type (
	// InventoryModel is an interface to be customized, add more methods here,
	// and implement the added methods in customInventoryModel.
	InventoryModel interface {
		inventoryModel
		// 全他妈 crud
		ExecWithTransaction(ctx context.Context, fn func(context.Context, sqlx.Session) error) error
		FreezeStockAtomic(ctx context.Context, id int64, count int64) error
		UnfreezeStockAtomic(ctx context.Context, id int64, count int64) error
		ConfirmFrozenToSold(ctx context.Context, id int64, count int64) error
		CancleSold(ctx context.Context, id int64, count int64) error
		DecrStock(ctx context.Context, id int64, count int64) error
		IncrStock(ctx context.Context, id int64, count int64) error
		FreezeWithSession(ctx context.Context, session sqlx.Session, id, count int64) error
		UnfreezeWithSession(ctx context.Context, session sqlx.Session, id, count int64) error
		ConfirmWithSession(ctx context.Context, session sqlx.Session, id, count int64) error
		CancleSoldWithSession(ctx context.Context, session sqlx.Session, id int64, count int64) error
		DecrWithSession(ctx context.Context, session sqlx.Session, id, count int64) error
		IncrWithSession(ctx context.Context, session sqlx.Session, id, count int64) error
		DecrWithSessionByMerchant(ctx context.Context, session sqlx.Session, id, merchantId, count int64) error
		IncrWithSessionByMerchant(ctx context.Context, session sqlx.Session, id, merchantId, count int64) error
		FindOneWithNoCache(ctx context.Context, productId int64) (*Inventory, error)
		InsertWithNoCache(ctx context.Context, data *Inventory) (sql.Result, error)
	}

	customInventoryModel struct {
		*defaultInventoryModel
	}
)

// NewInventoryModel returns a model for the database table.
func NewInventoryModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) InventoryModel {
	return &customInventoryModel{
		defaultInventoryModel: newInventoryModel(conn, c, opts...),
	}
}

func (m *customInventoryModel) ExecWithTransaction(ctx context.Context, fn func(context.Context, sqlx.Session) error) error {
	return m.TransactCtx(ctx, fn)
}

func (m *customInventoryModel) FreezeStockAtomic(ctx context.Context, id int64, count int64) error {
	if count <= 0 {
		return ErrInvalidParam
	}
	q := fmt.Sprintf(
		"UPDATE %s SET stock = stock - ?, frozen_stock = frozen_stock + ? WHERE product_id = ? AND stock >= ?",
		m.table,
	)
	res, err := m.ExecNoCacheCtx(ctx, q, count, count, id, count)
	if err != nil {
		return err
	}
	return ensureRows(res)
}

func (m *customInventoryModel) UnfreezeStockAtomic(ctx context.Context, id int64, count int64) error {
	if count <= 0 {
		return ErrInvalidParam
	}
	q := fmt.Sprintf(
		"UPDATE %s SET stock = stock + ?, frozen_stock = frozen_stock - ? WHERE product_id = ? AND frozen_stock >= ?",
		m.table,
	)
	res, err := m.ExecNoCacheCtx(ctx, q, count, count, id, count)
	if err != nil {
		return err
	}
	return ensureRows(res)
}

func (m *customInventoryModel) ConfirmFrozenToSold(ctx context.Context, id int64, count int64) error {
	if count <= 0 {
		return ErrInvalidParam
	}
	q := fmt.Sprintf(
		"UPDATE %s SET sold = sold + ?, frozen_stock = frozen_stock - ? WHERE product_id = ? AND frozen_stock >= ?",
		m.table,
	)
	res, err := m.ExecNoCacheCtx(ctx, q, count, count, id, count)
	if err != nil {
		return err
	}
	return ensureRows(res)
}

func (m *customInventoryModel) CancleSold(ctx context.Context, id int64, count int64) error {
	if count <= 0 {
		return ErrInvalidParam
	}
	q := fmt.Sprintf(
		"UPDATE %s SET sold = sold - ?, stock = stock + ? WHERE product_id = ? AND sold >= ?",
		m.table,
	)
	res, err := m.ExecNoCacheCtx(ctx, q, count, count, id, count)
	if err != nil {
		return err
	}
	return ensureRows(res)
}

func (m *customInventoryModel) DecrStock(ctx context.Context, id int64, count int64) error {
	if count <= 0 {
		return ErrInvalidParam
	}
	q := fmt.Sprintf(
		"UPDATE %s SET stock = stock - ? WHERE product_id = ? AND stock >= ?",
		m.table,
	)
	res, err := m.ExecNoCacheCtx(ctx, q, count, id, count)
	if err != nil {
		return err
	}
	return ensureRows(res)
}

func (m *customInventoryModel) IncrStock(ctx context.Context, id int64, count int64) error {
	if count <= 0 {
		return ErrInvalidParam
	}
	q := fmt.Sprintf(
		"UPDATE %s SET stock = stock + ? WHERE product_id = ?",
		m.table,
	)
	res, err := m.ExecNoCacheCtx(ctx, q, count, id)
	if err != nil {
		return err
	}
	return ensureRows(res)
}

func (m *customInventoryModel) FreezeWithSession(ctx context.Context, session sqlx.Session, id, count int64) error {
	if count <= 0 {
		return ErrInvalidParam
	}
	q := fmt.Sprintf(
		"UPDATE %s SET stock = stock - ?, frozen_stock = frozen_stock + ? WHERE product_id = ? AND stock >= ?",
		m.table,
	)
	res, err := session.ExecCtx(ctx, q, count, count, id, count)
	if err != nil {
		return err
	}
	return ensureRows(res)
}

func (m *customInventoryModel) UnfreezeWithSession(ctx context.Context, session sqlx.Session, id, count int64) error {
	if count <= 0 {
		return ErrInvalidParam
	}
	q := fmt.Sprintf(
		"UPDATE %s SET stock = stock + ?, frozen_stock = frozen_stock - ? WHERE product_id = ? AND frozen_stock >= ?",
		m.table,
	)
	res, err := session.ExecCtx(ctx, q, count, count, id, count)
	if err != nil {
		return err
	}
	return ensureRows(res)
}

func (m *customInventoryModel) ConfirmWithSession(ctx context.Context, session sqlx.Session, id, count int64) error {
	if count <= 0 {
		return ErrInvalidParam
	}
	q := fmt.Sprintf(
		"UPDATE %s SET sold = sold + ?, frozen_stock = frozen_stock - ? WHERE product_id = ? AND frozen_stock >= ?",
		m.table,
	)
	res, err := session.ExecCtx(ctx, q, count, count, id, count)
	if err != nil {
		return err
	}
	return ensureRows(res)
}

func (m *customInventoryModel) CancleSoldWithSession(ctx context.Context, session sqlx.Session, id int64, count int64) error {
	if count <= 0 {
		return ErrInvalidParam
	}
	q := fmt.Sprintf(
		"UPDATE %s SET sold = sold - ?, stock = stock + ? WHERE product_id = ? AND sold >= ?",
		m.table,
	)
	res, err := session.ExecCtx(ctx, q, count, count, id, count)
	if err != nil {
		return err
	}
	return ensureRows(res)
}

func (m *customInventoryModel) DecrWithSession(ctx context.Context, session sqlx.Session, id, count int64) error {
	if count <= 0 {
		return ErrInvalidParam
	}
	q := fmt.Sprintf(
		"UPDATE %s SET stock = stock - ? WHERE product_id = ? AND stock >= ?",
		m.table,
	)
	res, err := session.ExecCtx(ctx, q, count, id, count)
	if err != nil {
		return err
	}
	return ensureRows(res)
}

func (m *customInventoryModel) IncrWithSession(ctx context.Context, session sqlx.Session, id, count int64) error {
	if count <= 0 {
		return ErrInvalidParam
	}
	q := fmt.Sprintf(
		"UPDATE %s SET stock = stock + ? WHERE product_id = ?",
		m.table,
	)
	res, err := session.ExecCtx(ctx, q, count, id)
	if err != nil {
		return err
	}
	return ensureRows(res)
}

func (m *customInventoryModel) DecrWithSessionByMerchant(ctx context.Context, session sqlx.Session, id, merchantId, count int64) error {
	if count <= 0 || merchantId <= 0 {
		return ErrInvalidParam
	}
	q := fmt.Sprintf(
		"UPDATE %s SET stock = stock - ? WHERE product_id = ? AND merchant_id = ? AND stock >= ?",
		m.table,
	)
	res, err := session.ExecCtx(ctx, q, count, id, merchantId, count)
	if err != nil {
		return err
	}
	return ensureRows(res)
}

func (m *customInventoryModel) IncrWithSessionByMerchant(ctx context.Context, session sqlx.Session, id, merchantId, count int64) error {
	if count <= 0 || merchantId <= 0 {
		return ErrInvalidParam
	}
	q := fmt.Sprintf(
		"UPDATE %s SET stock = stock + ? WHERE product_id = ? AND merchant_id = ?",
		m.table,
	)
	res, err := session.ExecCtx(ctx, q, count, id, merchantId)
	if err != nil {
		return err
	}
	return ensureRows(res)
}

func ensureRows(res sql.Result) error {
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrRowsAffectedIsZero
	}
	return nil
}

func (m *customInventoryModel) FindOneWithNoCache(ctx context.Context, productId int64) (*Inventory, error) {
	query := fmt.Sprintf("select %s from %s where `product_id` = ? limit 1", inventoryRows, m.table)
	var resp Inventory
	err := m.QueryRowNoCacheCtx(ctx, &resp, query, productId)
	switch err {
	case nil:
		return &resp, nil
	case sqlc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}


func (m *customInventoryModel) InsertWithNoCache(ctx context.Context, data *Inventory) (sql.Result, error) {
	query := fmt.Sprintf("insert into %s (%s) values (?, ?, ?, ?, ?)", m.table, inventoryRowsExpectAutoSet)
	ret, err := m.ExecNoCacheCtx(ctx, query, data.ProductId, data.MerchantId, data.Stock, data.Sold, data.ForzenStock)
	return ret, err
}