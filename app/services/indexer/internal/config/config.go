package config

import (
	"github.com/zeromicro/go-zero/core/logx"
)

type Config struct {
	LogConf     logx.LogConf
	KafkaConf   KafkaConf
	ElasticConf ElasticConf
	Embedding   EmbeddingConf
}

type KafkaConf struct {
	Brokers              []string
	Group                string
	ProductsTopic        string
	ProductCategoryTopic string
}

type ElasticConf struct {
	Addresses          []string
	Username           string
	Password           string
	IndexName          string
	EmbeddingDimension int
}

type EmbeddingConf struct {
	BaseURL string
	APIKey  string
	Model   string
}
