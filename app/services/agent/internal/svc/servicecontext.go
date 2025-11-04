package svc

import (
	"context"

	"NatsumeAI/app/services/agent/internal/config"
	"NatsumeAI/app/services/product/product"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config

	ChatModel   *ark.ChatModel
	ProductRpc  product.ProductServiceClient
	productConn zrpc.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.MustSetup(c.LogConf)

	sc := &ServiceContext{Config: c}

	cm, err := ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
		BaseURL: c.ChatModel.BaseUrl,
		APIKey:  c.ChatModel.APIKey,
		Model:   c.ChatModel.Model,
	})
	if err != nil {
		logx.Errorw("init ark chat model failed", logx.Field("err", err))
	} else {
		sc.ChatModel = cm
		logx.Infow("ark chat model initialized")
	}

	if c.ProductRpc.Target != "" {
		conn := zrpc.MustNewClient(c.ProductRpc)
		sc.productConn = conn
		sc.ProductRpc = product.NewProductServiceClient(conn.Conn())
		logx.Infow("product rpc client initialized")
	} else {
		logx.Infof("empty product rpc target, product client disabled")
	}

	return sc
}
