// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package agent

import (
	"context"

	"NatsumeAI/app/api/agent/internal/logic/helper"
	"NatsumeAI/app/api/agent/internal/svc"
	"NatsumeAI/app/api/agent/internal/types"
	"NatsumeAI/app/common/util"
	"NatsumeAI/app/services/agent/agentservice"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChatLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatLogic {
	return &ChatLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChatLogic) Chat(req *types.ChatRequest) (resp *types.ChatResponse, err error) {
	userId, err := util.UserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}
	in := agentservice.ChatReq{
		UserId: userId,
		Query: req.Query,
	}
	res, err := l.svcCtx.AgentRpc.Chat(l.ctx, &in)
	resp = helper.ToChatResponse(res)
	
	return
}
 