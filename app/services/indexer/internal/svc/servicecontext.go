package svc

import (
	"NatsumeAI/app/services/indexer/internal/config"
	"context"
	"strings"

	embeddingark "github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config   config.Config
	ESClient *elasticsearch.Client
	Embedder *embeddingark.Embedder
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.MustSetup(c.LogConf)

	var esClient *elasticsearch.Client
	var embedder *embeddingark.Embedder
	if len(c.ElasticConf.Addresses) > 0 {
		client, err := elasticsearch.NewClient(elasticsearch.Config{
			Addresses: c.ElasticConf.Addresses,
			Username:  c.ElasticConf.Username,
			Password:  c.ElasticConf.Password,
		})
		if err != nil {
			logx.Errorw("init elasticsearch client failed", logx.Field("err", err))
		} else {
			esClient = client
			logx.Infow("elasticsearch client initialized", logx.Field("addresses", c.ElasticConf.Addresses))
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
			embedder = emb
			logx.Infow("embedding model initialized", logx.Field("model", c.Embedding.Model))
		}
	} else {
		logx.Infow("embedding client disabled, missing model or api key")
	}

	return &ServiceContext{
		Config:   c,
		ESClient: esClient,
		Embedder: embedder,
	}
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
	return 2048
}
