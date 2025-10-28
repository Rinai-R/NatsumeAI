package svc

import (
	"NatsumeAI/app/services/agent/internal/config"
	"context"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
    Config config.Config

    ChatModel *ark.ChatModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.MustSetup(c.LogConf)
    sc := &ServiceContext{Config: c}

    cm, err := ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
		BaseURL: c.ChatModel.BaseUrl,
        APIKey: c.ChatModel.APIKey,
        Model:  c.ChatModel.Model,
    })
    if err != nil {
        logx.Errorw("init ark chat model failed", logx.Field("err", err))
	} else {
        sc.ChatModel = cm
        logx.Infow("ark chat model initialized")
    }
    
    return sc
}
