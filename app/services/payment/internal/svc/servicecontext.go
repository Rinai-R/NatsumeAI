package svc

import (
	"time"

	"NatsumeAI/app/common/snowflake"
	paymentdal "NatsumeAI/app/dal/payment"
	"NatsumeAI/app/services/order/order"
	"NatsumeAI/app/services/payment/internal/config"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config

	DB            sqlx.SqlConn
	PaymentOrders paymentdal.PaymentOrdersModel

	OrderRpc    order.OrderServiceClient
	AsynqClient *asynq.Client

	PaymentTTL time.Duration
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.MustSetup(c.LogConf)

	db := sqlx.NewMysql(c.MysqlConf.DataSource)
	pOrders := paymentdal.NewPaymentOrdersModel(db, c.CacheConf)

	orderCli := order.NewOrderServiceClient(zrpc.MustNewClient(c.OrderRpc).Conn())

	asynqAddr := c.AsynqConf.Addr
	if asynqAddr == "" {
		asynqAddr = c.RedisConf.Host
	}
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: asynqAddr})

	ttl := time.Duration(c.PaymentTimeoutMinutes) * time.Minute
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	if c.SnowflakeNode > 0 {
		if err := snowflake.SetNodeID(c.SnowflakeNode); err != nil {
			logx.Errorf("failed to set snowflake node id: %v", err)
		}
	}

	return &ServiceContext{
		Config:        c,
		DB:            db,
		PaymentOrders: pOrders,
		OrderRpc:      orderCli,
		AsynqClient:   asynqClient,
		PaymentTTL:    ttl,
	}
}
