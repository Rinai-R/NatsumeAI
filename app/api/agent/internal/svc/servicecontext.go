// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"NatsumeAI/app/api/agent/internal/config"
	"NatsumeAI/app/common/middleware"
	"NatsumeAI/app/services/agent/agentservice"
	"NatsumeAI/app/services/auth/authservice"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
    Config         config.Config
    AuthMiddleware rest.Middleware
    CasbinMiddleware rest.Middleware
    AgentRpc agentservice.AgentService
}

func NewServiceContext(c config.Config) *ServiceContext {
    return &ServiceContext{
        Config:         c,
        AuthMiddleware: middleware.NewAuthMiddleware(
            authservice.NewAuthService(zrpc.MustNewClient(c.AuthRpc))).Handle,
        CasbinMiddleware: middleware.NewCasbinMiddleware(
            c.CasbinMiddleware.MustNewDistributedEnforcer()).Handle,
        AgentRpc: agentservice.NewAgentService(zrpc.MustNewClient(c.AgentRpc)),
    }
}
