package svc

import (
    "database/sql"
    "time"

    orderdal "NatsumeAI/app/dal/order"
    couponsvc "NatsumeAI/app/services/coupon/couponservice"
    invsvc "NatsumeAI/app/services/inventory/inventoryservice"
    "NatsumeAI/app/services/order/internal/config"
    prodsvc "NatsumeAI/app/services/product/productservice"

    "github.com/hibiken/asynq"
    "github.com/segmentio/kafka-go"
    "github.com/zeromicro/go-zero/core/logx"
    "github.com/zeromicro/go-zero/core/stores/sqlx"
    "github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
    Config config.Config

    DB       sqlx.SqlConn
    RawDB    *sql.DB
    Preorder orderdal.OrderPreordersModel
    PreItm   orderdal.OrderPreorderItemsModel
    Orders   orderdal.OrdersModel
    OrdItm   orderdal.OrderItemsModel

    Inventory invsvc.InventoryService
    Coupon    couponsvc.CouponService
    Product   prodsvc.ProductService

    AsynqClient *asynq.Client

    KafkaWriter *kafka.Writer

    // Global preorder TTL for computing ExpireAt and scheduling delays
    PreorderTTL time.Duration
}

func NewServiceContext(c config.Config) *ServiceContext {
    logx.MustSetup(c.LogConf)
    db := sqlx.NewMysql(c.MysqlConf.DataSource)
    raw, _ := db.RawDB()
    invCli := invsvc.NewInventoryService(zrpc.MustNewClient(c.InventoryRpc))
    var coupCli couponsvc.CouponService
    if c.CouponRpc.Target != "" {
        coupCli = couponsvc.NewCouponService(zrpc.MustNewClient(c.CouponRpc))
    }
    var prodCli prodsvc.ProductService
    if c.ProductRpc.Target != "" {
        prodCli = prodsvc.NewProductService(zrpc.MustNewClient(c.ProductRpc))
    }
    asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: c.AsynqConf.Addr})

    // Reusable Kafka writer to reduce per-send overhead and latency
    var kw *kafka.Writer
    if len(c.KafkaConf.Broker) > 0 && c.KafkaConf.PreOrderTopic != "" {
        kw = &kafka.Writer{
            Addr:                    kafka.TCP(c.KafkaConf.Broker...),
            Topic:                   c.KafkaConf.PreOrderTopic,
            RequiredAcks:            kafka.RequireOne,
            Balancer:                &kafka.LeastBytes{},
            AllowAutoTopicCreation:  true,
            BatchTimeout:            5 * time.Millisecond,
        }
    }

    // Compute preorder TTL (default 30m if missing)
    ttl := time.Duration(c.PreorderTTLMinutes) * time.Minute
    if ttl <= 0 {
        ttl = 30 * time.Minute
    }

    sc := &ServiceContext{
        Config:     c,
        DB:         db,
        RawDB:      raw,
        Preorder:   orderdal.NewOrderPreordersModel(db, c.CacheConf),
        PreItm:     orderdal.NewOrderPreorderItemsModel(db, c.CacheConf),
        Orders:     orderdal.NewOrdersModel(db, c.CacheConf),
        OrdItm:     orderdal.NewOrderItemsModel(db, c.CacheConf),
        Inventory:  invCli,
        Coupon:     coupCli,
        Product:    prodCli,
        AsynqClient: asynqClient,
        KafkaWriter: kw,
        PreorderTTL: ttl,
    }

    return sc
}
