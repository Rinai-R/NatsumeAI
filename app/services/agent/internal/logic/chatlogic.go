package logic

import (
	"context"

	"NatsumeAI/app/services/agent/agent"
	internalagent "NatsumeAI/app/services/agent/internal/agent/chat"
	"NatsumeAI/app/services/agent/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChatLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatLogic {
	return &ChatLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 对话接口（非流式），响应中包含商品推荐
func (l *ChatLogic) Chat(in *agent.ChatReq) (*agent.ChatResp, error) {
	chatAgent := internalagent.NewAgent(l.ctx, l.svcCtx)
	resp, err := chatAgent.Chat(l.ctx, in)
	if err != nil {
		l.Logger.Errorf("chat agent failed: %v", err)
		return &agent.ChatResp{
			StatusCode: 500,
			StatusMsg:  "agent error",
			Answer:     "抱歉，助手出现了一点问题，请稍后再试。",
		}, nil
	}
	return resp, nil
}
