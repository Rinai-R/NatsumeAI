package svc

import (
    "database/sql"

    orderdal "NatsumeAI/app/dal/order"
    couponsvc "NatsumeAI/app/services/coupon/couponservice"
    invsvc "NatsumeAI/app/services/inventory/inventoryservice"
    prodsvc "NatsumeAI/app/services/product/productservice"
    "NatsumeAI/app/services/order/internal/config"

    "github.com/hibiken/asynq"
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
}

func NewServiceContext(c config.Config) *ServiceContext {
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
    // asynq client — use RedisConf
    asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: c.RedisConf.Host, Password: c.RedisConf.Pass})

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
    }

    return sc
}
