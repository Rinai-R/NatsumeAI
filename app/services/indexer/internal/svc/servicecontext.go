package svc

import (
	"NatsumeAI/app/services/indexer/internal/config"
	"NatsumeAI/app/services/indexer/internal/es"
	"context"
	"errors"
	"strings"

	embeddingark "github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config           config.Config
	ESClient         *elasticsearch.Client
	Embedder         *embeddingark.Embedder
	vectorIndexReady bool
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.MustSetup(c.LogConf)

	ctx := &ServiceContext{
		Config: c,
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
			ctx.ESClient = client
			logx.Infow("elasticsearch client initialized", logx.Field("addresses", c.ElasticConf.Addresses))
			if info, err := es.EnsureProductIndex(context.Background(), client, es.ProductIndexParams{
				IndexName:        ctx.ProductIndexName(),
				EmbeddingDims:    ctx.EmbeddingDimension(),
				NumberOfShards:   c.ElasticConf.Shards,
				NumberOfReplicas: c.ElasticConf.Replicas,
			}); err != nil {
				if errors.Is(err, es.ErrIncompatibleEmbeddingMapping) {
					logx.Errorw("product index embedding mapping incompatible",
						logx.Field("index", ctx.ProductIndexName()),
						logx.Field("err", err))
				} else {
					logx.Errorw("ensure product index failed",
						logx.Field("index", ctx.ProductIndexName()),
						logx.Field("err", err))
				}
			} else {
				ctx.vectorIndexReady = info.SupportsVector
			}
		}
	} else {
		logx.Infow("elasticsearch client disabled, no addresses configured")
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
			ctx.Embedder = emb
			logx.Infow("embedding model initialized", logx.Field("model", c.Embedding.Model))
		}
	} else {
		logx.Infow("embedding client disabled, missing model or api key")
	}

	return ctx
}

func (s *ServiceContext) ProductIndexName() string {
	if idx := strings.TrimSpace(s.Config.ElasticConf.IndexName); idx != "" {
		return idx
	}
	return "products"
}

func (s *ServiceContext) VectorIndexEnabled() bool {
	return s.vectorIndexReady
}

func (s *ServiceContext) EmbeddingDimension() int {
	if s.Config.ElasticConf.EmbeddingDimension > 0 {
		return s.Config.ElasticConf.EmbeddingDimension
	}
	return 2048
}
