package logic

import (
	"context"

	"NatsumeAI/app/services/agent/agent"
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
	// todo: add your logic here and delete this line

	return &agent.ChatResp{}, nil
}
