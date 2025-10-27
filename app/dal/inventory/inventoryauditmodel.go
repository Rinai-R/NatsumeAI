package inventory

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ InventoryAuditModel = (*customInventoryAuditModel)(nil)

type (
	// InventoryAuditModel is an interface to be customized, add more methods here,
	// and implement the added methods in customInventoryAuditModel.
    InventoryAuditModel interface {
        inventoryAuditModel
        InsertWithSession(ctx context.Context, session sqlx.Session, data *InventoryAudit) (sql.Result, error)
        UpdateStatusWithSession(ctx context.Context, session sqlx.Session, orderId, productId int64, status string) error
        FindWithOrderId(ctx context.Context, orderId int64) (InventoryAudit, error)
        // ExistsWithSession checks if an audit record exists for (orderId, productId)
        // within the given session/transaction.
        ExistsWithSession(ctx context.Context, session sqlx.Session, orderId, productId int64) (bool, error)
        // GetStatusWithSession returns (status, true, nil) if audit exists, ("", false, nil) if not found.
        GetStatusWithSession(ctx context.Context, session sqlx.Session, orderId, productId int64) (string, bool, error)
    }

	customInventoryAuditModel struct {
		*defaultInventoryAuditModel
	}
)

// NewInventoryAuditModel returns a model for the database table.
func NewInventoryAuditModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) InventoryAuditModel {
    return &customInventoryAuditModel{
        defaultInventoryAuditModel: newInventoryAuditModel(conn, c, opts...),
    }
}

// audit 用个缓存会减少开销
func (m *customInventoryAuditModel) InsertWithSession(ctx context.Context, session sqlx.Session, data *InventoryAudit) (sql.Result, error) {
	query := fmt.Sprintf("insert into %s (%s) values (?, ?, ?, ?)", m.table, inventoryAuditRowsExpectAutoSet)
	res, err := session.ExecCtx(ctx, query, data.OrderId, data.ProductId, data.Quantity, data.Status)
	if err != nil {
		return nil, err
	}
	if err := ensureRows(res); err != nil {
		return nil, err
	}

	keys := []string{
		fmt.Sprintf("%s%v:%v", cacheInventoryAuditOrderIdProductIdPrefix, data.OrderId, data.ProductId),
	}
	if id, err := res.LastInsertId(); err == nil {
		keys = append(keys, fmt.Sprintf("%s%v", cacheInventoryAuditIdPrefix, id))
	} else if data.Id > 0 {
		keys = append(keys, fmt.Sprintf("%s%v", cacheInventoryAuditIdPrefix, data.Id))
	}
	if err := m.DelCacheCtx(ctx, keys...); err != nil {
		return res, err
	}

	return res, nil
}

func (m *customInventoryAuditModel) ExistsWithSession(ctx context.Context, session sqlx.Session, orderId, productId int64) (bool, error) {
    var id int64
    q := fmt.Sprintf("select `id` from %s where `order_id` = ? and `product_id` = ? limit 1", m.table)
    if err := session.QueryRowCtx(ctx, &id, q, orderId, productId); err != nil {
        if err == sql.ErrNoRows || err == sqlx.ErrNotFound {
            return false, nil
        }
        return false, err
    }
    return id > 0, nil
}

func (m *customInventoryAuditModel) GetStatusWithSession(ctx context.Context, session sqlx.Session, orderId, productId int64) (string, bool, error) {
    var status string
    q := fmt.Sprintf("select `status` from %s where `order_id` = ? and `product_id` = ? limit 1", m.table)
    if err := session.QueryRowCtx(ctx, &status, q, orderId, productId); err != nil {
        if err == sql.ErrNoRows || err == sqlx.ErrNotFound {
            return "", false, nil
        }
        return "", false, err
    }
    return status, true, nil
}

func (m *customInventoryAuditModel) UpdateStatusWithSession(ctx context.Context, session sqlx.Session, orderId, productId int64, status string) error {
	if status != AUDIT_CANCLLED && status != AUDIT_CONFIRMED && status != AUDIT_PENDING {
		return ErrInvalidParam
	}
	data, err := m.FindOneByOrderIdProductId(ctx, orderId, productId)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("update %s set `status` = ? where `order_id` = ? and `product_id` = ?", m.table)
	res, err := session.ExecCtx(ctx, query, status, orderId, productId)
	if err != nil {
		return err
	}
	if err := ensureRows(res); err != nil {
		return err
	}

	keys := []string{
		fmt.Sprintf("%s%v:%v", cacheInventoryAuditOrderIdProductIdPrefix, data.OrderId, data.ProductId),
		fmt.Sprintf("%s%v", cacheInventoryAuditIdPrefix, data.Id),
	}
	return m.DelCacheCtx(ctx, keys...)
}

func (m *customInventoryAuditModel) FindWithOrderId(ctx context.Context, orderId int64) (InventoryAudit, error) {
	var audit InventoryAudit
	query := fmt.Sprintf("select %s from %s where `order_id` = ?", inventoryAuditRows, m.table)
	err := m.QueryRowsNoCacheCtx(ctx, &audit, query, orderId)
	return audit, err
}
