package chat

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"NatsumeAI/app/services/agent/agent"
	"NatsumeAI/app/services/agent/internal/agent/intent"
	"NatsumeAI/app/services/agent/internal/agent/tools"
	"NatsumeAI/app/services/agent/internal/svc"
	productpb "NatsumeAI/app/services/product/product"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	defaultSearchLimit     int32 = 5
	maxToolIterations            = 1
	maxToolCallsPerSession       = 3
)

type Agent struct {
	log          logx.Logger
	model        *ark.ChatModel
	productCli   productpb.ProductServiceClient
	intent       *intent.Classifier
	toolInfos    []*schema.ToolInfo
	toolExecutor *tools.Executor

	toolMu    sync.RWMutex
	toolModel model.ToolCallingChatModel
}

func NewAgent(ctx context.Context, svcCtx *svc.ServiceContext) *Agent {
	a := &Agent{
		log:        logx.WithContext(ctx),
		model:      svcCtx.ChatModel,
		productCli: svcCtx.ProductRpc,
	}

	if svcCtx.ChatModel != nil {
		classifier, err := intent.NewClassifier(ctx, a.log, svcCtx.ChatModel)
		if err != nil {
			a.log.Errorf("init intent classifier failed: %v", err)
		} else {
			a.intent = classifier
		}
	}

	a.toolInfos = tools.BuildToolInfos()
	if svcCtx.ProductRpc != nil {
		a.toolExecutor = tools.NewExecutor(a.log, svcCtx.ProductRpc, defaultSearchLimit)
	}

	return a
}

func (a *Agent) Chat(ctx context.Context, req *agent.ChatReq) (*agent.ChatResp, error) {
	resp := &agent.ChatResp{
		StatusCode: 0,
		StatusMsg:  "ok",
	}

	if req == nil {
		resp.StatusCode = 400
		resp.StatusMsg = "invalid request"
		return resp, nil
	}

	intentStart := time.Now()
	// 先推断意图
	decision, err := a.decideIntent(ctx, req)
	if err != nil {
		a.log.Errorf("intent decision failed: %v", err)
		return &agent.ChatResp{
			StatusCode: 500,
			StatusMsg:  "intent decision error",
			Answer:     "抱歉，助手暂时无法理解本次需求，请稍后再试。",
		}, nil
	}
	a.log.Infof("intent decision success in %s", time.Since(intentStart))
	if decision != nil && !decision.ShouldRecommend() {
		resp.Answer = a.noRecommendationAnswer(ctx, req)
		return resp, nil
	}

	if a.model == nil {
		return &agent.ChatResp{
			StatusCode: 500,
			StatusMsg:  "chat model unavailable",
			Answer:     "抱歉，推荐服务暂时不可用，请稍后再试。",
		}, nil
	}
	if a.productCli == nil {
		resp.StatusCode = 500
		resp.StatusMsg = "product rpc unavailable"
		resp.Answer = "抱歉，商品服务暂时不可用，请稍后再试。"
		return resp, nil
	}

	result, err := a.runToolDrivenChat(ctx, req, decision)
	if err != nil {
		a.log.Errorf("tool driven chat failed: %v", err)
		return &agent.ChatResp{
			StatusCode: 500,
			StatusMsg:  "agent execution error",
			Answer:     "抱歉，助手处理本次请求时出现问题，请稍后再试。",
		}, nil
	}
	return result, nil
}

func (a *Agent) decideIntent(ctx context.Context, req *agent.ChatReq) (*intent.Decision, error) {
	if req == nil {
		return nil, fmt.Errorf("empty request")
	}
	if a.intent == nil {
		return nil, fmt.Errorf("intent classifier unavailable")
	}

	res, err := a.intent.Analyze(ctx, intent.Input{
		UserID: req.GetUserId(),
		Query:  req.GetQuery(),
	})
	if err != nil {
		if strings.Contains(err.Error(), "intent decision tool payload missing") {
			a.log.Infof("intent decision returned empty tool payload, fallback to recommend: %v", err)
			return &intent.Decision{Intent: intent.IntentRecommend}, nil
		}
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("empty intention result")
	}

	return res, nil
}

func (a *Agent) ensureToolModel() (model.ToolCallingChatModel, error) {
	if a == nil {
		return nil, fmt.Errorf("agent is nil")
	}
	if a.model == nil {
		return nil, fmt.Errorf("chat model unavailable")
	}

	toolCapable, ok := any(a.model).(model.ToolCallingChatModel)
	if !ok {
		return nil, fmt.Errorf("chat model does not support tool calling")
	}

	a.toolMu.RLock()
	if a.toolModel != nil {
		defer a.toolMu.RUnlock()
		return a.toolModel, nil
	}
	a.toolMu.RUnlock()

	a.toolMu.Lock()
	defer a.toolMu.Unlock()

	if a.toolModel != nil {
		return a.toolModel, nil
	}

	modelWithTools, err := toolCapable.WithTools(a.toolInfos)
	if err != nil {
		return nil, err
	}

	a.toolModel = modelWithTools
	return a.toolModel, nil
}
