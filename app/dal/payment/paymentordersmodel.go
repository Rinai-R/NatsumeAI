package payment

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ PaymentOrdersModel = (*customPaymentOrdersModel)(nil)

type (
	// PaymentOrdersModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPaymentOrdersModel.
	PaymentOrdersModel interface {
		paymentOrdersModel
		UpdateStatus(ctx context.Context, paymentId int64, fromStatus []string, toStatus string) (bool, error)
	}

	customPaymentOrdersModel struct {
		*defaultPaymentOrdersModel
	}
)

// NewPaymentOrdersModel returns a model for the database table.
func NewPaymentOrdersModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) PaymentOrdersModel {
	return &customPaymentOrdersModel{
		defaultPaymentOrdersModel: newPaymentOrdersModel(conn, c, opts...),
	}
}

func (m *customPaymentOrdersModel) UpdateStatus(ctx context.Context, paymentId int64, fromStatus []string, toStatus string) (bool, error) {
	if len(fromStatus) == 0 {
		return false, fmt.Errorf("fromStatus must not be empty")
	}

	record, err := m.FindOne(ctx, paymentId)
	if err != nil {
		return false, err
	}

	paymentOrdersOrderIdKey := fmt.Sprintf("%s%v", cachePaymentOrdersOrderIdPrefix, record.OrderId)
	paymentOrdersPaymentIdKey := fmt.Sprintf("%s%v", cachePaymentOrdersPaymentIdPrefix, paymentId)
	paymentOrdersPaymentNoKey := fmt.Sprintf("%s%v", cachePaymentOrdersPaymentNoPrefix, record.PaymentNo)

	args := make([]any, 0, len(fromStatus)+2)
	args = append(args, toStatus)
	for _, s := range fromStatus {
		args = append(args, s)
	}
	args = append(args, paymentId)

	query := fmt.Sprintf("update %s set `status` = ?, `updated_at` = current_timestamp where `status` in (%s) and `payment_id` = ?", m.tableName(), placeholders(len(fromStatus)))

	result, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, query, args...)
	}, paymentOrdersOrderIdKey, paymentOrdersPaymentIdKey, paymentOrdersPaymentNoKey)
	if err != nil {
		return false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}

	var builder strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteByte('?')
	}
	return builder.String()
}
