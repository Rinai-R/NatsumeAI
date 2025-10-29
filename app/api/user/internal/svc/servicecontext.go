// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
    "NatsumeAI/app/api/user/internal/config"
    "NatsumeAI/app/common/middleware"
    "NatsumeAI/app/services/auth/authservice"
    "NatsumeAI/app/services/user/userservice"

    "github.com/zeromicro/go-zero/core/logx"
    "github.com/zeromicro/go-zero/rest"
    "github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
    Config         config.Config
    AuthMiddleware rest.Middleware
    CasbinMiddleware rest.Middleware
    UserRpc userservice.UserService
    AuthRpc authservice.AuthService
}

func NewServiceContext(c config.Config) *ServiceContext {
    logx.MustSetup(c.LogConf)
    return &ServiceContext{
        Config:         c,
        AuthMiddleware: middleware.NewAuthMiddleware(authservice.NewAuthService(zrpc.MustNewClient(c.AuthRpc))).Handle,
        CasbinMiddleware: middleware.NewCasbinMiddleware(c.CasbinMiddleware.MustNewDistributedEnforcer()).Handle,
        UserRpc: userservice.NewUserService(zrpc.MustNewClient(c.UserRpc)),
        AuthRpc: authservice.NewAuthService(zrpc.MustNewClient(c.AuthRpc)),
    }
}
