package svc

import (
	"NatsumeAI/app/common/consts/biz"
	"NatsumeAI/app/dal/product"
	"NatsumeAI/app/services/inventory/inventory"
	"NatsumeAI/app/services/inventory/inventoryservice"
	"NatsumeAI/app/services/product/internal/config"
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/bloom"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config
	InventoryRpc inventory.InventoryServiceClient
	ProductModel product.ProductsModel
	ProductCategoriesModel product.ProductCategoriesModel
	Bloom *bloom.Filter
}

func NewServiceContext(c config.Config) *ServiceContext {
	bf := bloom.New(redis.MustNewRedis(c.RedisConf), biz.PRODUCT_CHECK_BLOOM, biz.PRODUCT_CHECK_BLOOM_BIT)
	ProductsModel := product.NewProductsModel(sqlx.MustNewConn(c.MysqlConf), c.CacheConf)
	err := bloomPreheat(bf, ProductsModel)
	if err != nil {
		panic(err)
	}
	return &ServiceContext{
		Config: c,
		InventoryRpc: inventoryservice.NewInventoryService(zrpc.MustNewClient(c.InventoryRpc)),
		ProductModel: ProductsModel,
		ProductCategoriesModel:  product.NewProductCategoriesModel(sqlx.MustNewConn(c.MysqlConf), c.CacheConf),
		Bloom: bf,
	}
}

func bloomPreheat(bf *bloom.Filter, ProductsModel product.ProductsModel) error {
	ids, err := ProductsModel.FindAllProductId(context.TODO())
	if err != nil {
		return err
	}

	for _, id := range ids {
		err := bf.Add([]byte(strconv.Itoa(int(id))))
		if err != nil {
			return err
		}
	}
	return nil
}
