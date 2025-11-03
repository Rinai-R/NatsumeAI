package payment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

const paymentOrdersTable = "payment_orders"

type PaymentOrders struct {
	PaymentId      int64          `db:"payment_id"`
	PaymentNo      string         `db:"payment_no"`
	OrderId        int64          `db:"order_id"`
	UserId         int64          `db:"user_id"`
	Amount         int64          `db:"amount"`
	Currency       string         `db:"currency"`
	Channel        string         `db:"channel"`
	Status         string         `db:"status"`
	ChannelPayload sql.NullString `db:"channel_payload"`
	TimeoutAt      time.Time      `db:"timeout_at"`
	Extra          sql.NullString `db:"extra"`
	CreatedAt      time.Time      `db:"created_at"`
	UpdatedAt      time.Time      `db:"updated_at"`
}

type PaymentOrdersModel interface {
	Insert(ctx context.Context, data *PaymentOrders) (sql.Result, error)
	FindOne(ctx context.Context, paymentId int64) (*PaymentOrders, error)
	FindOneByPaymentNo(ctx context.Context, paymentNo string) (*PaymentOrders, error)
	FindOneByOrderId(ctx context.Context, orderId int64) (*PaymentOrders, error)
	UpdateStatus(ctx context.Context, id int64, fromStatus []string, toStatus string) (bool, error)
}

type defaultPaymentOrdersModel struct {
	conn sqlx.SqlConn
}

func NewPaymentOrdersModel(conn sqlx.SqlConn) PaymentOrdersModel {
	return &defaultPaymentOrdersModel{conn: conn}
}

func (m *defaultPaymentOrdersModel) Insert(ctx context.Context, data *PaymentOrders) (sql.Result, error) {
	query := fmt.Sprintf("insert into %s (payment_no, order_id, user_id, amount, currency, channel, status, channel_payload, timeout_at, extra) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", paymentOrdersTable)
	return m.conn.ExecCtx(ctx, query,
		data.PaymentNo,
		data.OrderId,
		data.UserId,
		data.Amount,
		data.Currency,
		data.Channel,
		data.Status,
		data.ChannelPayload,
		data.TimeoutAt,
		data.Extra,
	)
}

func (m *defaultPaymentOrdersModel) FindOne(ctx context.Context, paymentId int64) (*PaymentOrders, error) {
	query := fmt.Sprintf("select payment_id, payment_no, order_id, user_id, amount, currency, channel, status, channel_payload, timeout_at, extra, created_at, updated_at from %s where payment_id = ? limit 1", paymentOrdersTable)
	var po PaymentOrders
	err := m.conn.QueryRowCtx(ctx, &po, query, paymentId)
	if err != nil {
		return nil, err
	}
	return &po, nil
}

func (m *defaultPaymentOrdersModel) FindOneByPaymentNo(ctx context.Context, paymentNo string) (*PaymentOrders, error) {
	query := fmt.Sprintf("select payment_id, payment_no, order_id, user_id, amount, currency, channel, status, channel_payload, timeout_at, extra, created_at, updated_at from %s where payment_no = ? limit 1", paymentOrdersTable)
	var po PaymentOrders
	err := m.conn.QueryRowCtx(ctx, &po, query, paymentNo)
	if err != nil {
		return nil, err
	}
	return &po, nil
}

func (m *defaultPaymentOrdersModel) FindOneByOrderId(ctx context.Context, orderId int64) (*PaymentOrders, error) {
	query := fmt.Sprintf("select payment_id, payment_no, order_id, user_id, amount, currency, channel, status, channel_payload, timeout_at, extra, created_at, updated_at from %s where order_id = ? limit 1", paymentOrdersTable)
	var po PaymentOrders
	err := m.conn.QueryRowCtx(ctx, &po, query, orderId)
	if err != nil {
		return nil, err
	}
	return &po, nil
}

func (m *defaultPaymentOrdersModel) UpdateStatus(ctx context.Context, id int64, fromStatus []string, toStatus string) (bool, error) {
	if len(fromStatus) == 0 {
		return false, fmt.Errorf("fromStatus must not be empty")
	}
	args := make([]interface{}, 0, len(fromStatus)+2)
	args = append(args, toStatus)
	for _, s := range fromStatus {
		args = append(args, s)
	}
	args = append(args, id)
	query := fmt.Sprintf("update %s set status = ?, updated_at = current_timestamp where status in (%s) and payment_id = ?", paymentOrdersTable, placeholders(len(fromStatus)))
	res, err := m.conn.ExecCtx(ctx, query, args...)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	p := "?"
	for i := 1; i < n; i++ {
		p += ",?"
	}
	return p
}
