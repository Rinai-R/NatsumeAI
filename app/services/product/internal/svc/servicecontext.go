package svc

import (
	"NatsumeAI/app/common/consts/biz"
	"NatsumeAI/app/dal/product"
	"NatsumeAI/app/services/inventory/inventory"
	"NatsumeAI/app/services/inventory/inventoryservice"
	"NatsumeAI/app/services/product/internal/config"
	"context"
	"strconv"
	"strings"

	embeddingark "github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/zeromicro/go-zero/core/bloom"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config                 config.Config
	InventoryRpc           inventory.InventoryServiceClient
	ProductModel           product.ProductsModel
	ProductCategoriesModel product.ProductCategoriesModel
	Bloom                  *bloom.Filter
	ESClient               *elasticsearch.Client
	Embedder               *embeddingark.Embedder
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.MustSetup(c.LogConf)
	bf := bloom.New(redis.MustNewRedis(c.RedisConf), biz.PRODUCT_CHECK_BLOOM, biz.PRODUCT_CHECK_BLOOM_BIT)
	ProductsModel := product.NewProductsModel(sqlx.MustNewConn(c.MysqlConf), c.CacheConf)
	err := bloomPreheat(bf, ProductsModel)
	if err != nil {
		panic(err)
	}
	sc := &ServiceContext{
		Config:                 c,
		InventoryRpc:           inventoryservice.NewInventoryService(zrpc.MustNewClient(c.InventoryRpc)),
		ProductModel:           ProductsModel,
		ProductCategoriesModel: product.NewProductCategoriesModel(sqlx.MustNewConn(c.MysqlConf), c.CacheConf),
		Bloom:                  bf,
	}

	if len(c.ElasticConf.Addresses) > 0 {
		client, err := elasticsearch.NewClient(elasticsearch.Config{
			Addresses: c.ElasticConf.Addresses,
			Username:  c.ElasticConf.Username,
			Password:  c.ElasticConf.Password,
		})
		if err != nil {
			logx.Errorw("init elasticsearch client failed", logx.Field("err", err))
		} else {
			sc.ESClient = client
			logx.Infow("elasticsearch client initialized for product service", logx.Field("addresses", c.ElasticConf.Addresses))
		}
	}

	if c.Embedding.Model != "" && c.Embedding.APIKey != "" {
		emb, err := embeddingark.NewEmbedder(context.Background(), &embeddingark.EmbeddingConfig{
			BaseURL: c.Embedding.BaseURL,
			APIKey:  c.Embedding.APIKey,
			Model:   c.Embedding.Model,
		})
		if err != nil {
			logx.Errorw("init embedding model failed", logx.Field("err", err))
		} else {
			sc.Embedder = emb
			logx.Infow("embedding model initialized for product service", logx.Field("model", c.Embedding.Model))
		}
	} else {
		logx.Infow("embedding client disabled for product service, missing model or api key")
	}

	return sc
}

func bloomPreheat(bf *bloom.Filter, ProductsModel product.ProductsModel) error {
	ids, err := ProductsModel.FindAllProductId(context.TODO())
	if err != nil && err != product.ErrNotFound {
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

func (s *ServiceContext) ProductIndexName() string {
	if idx := strings.TrimSpace(s.Config.ElasticConf.IndexName); idx != "" {
		return idx
	}
	return "products"
}

func (s *ServiceContext) EmbeddingDimension() int {
	if s.Config.ElasticConf.EmbeddingDimension > 0 {
		return s.Config.ElasticConf.EmbeddingDimension
	}
	return 384
}

func (s *ServiceContext) HybridAlpha() float64 {
	alpha := s.Config.ElasticConf.Alpha
	if alpha <= 0 || alpha >= 1 {
		return 0.5
	}
	return alpha
}
