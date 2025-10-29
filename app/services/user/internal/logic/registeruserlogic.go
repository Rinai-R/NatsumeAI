package logic

import (
	"context"
	"strconv"
	"strings"

	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/common/snowflake"
	model "NatsumeAI/app/dal/user"
	"NatsumeAI/app/services/user/internal/svc"
	"NatsumeAI/app/services/user/user"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
)

type RegisterUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterUserLogic {
	return &RegisterUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RegisterUserLogic) RegisterUser(in *user.RegisterUserRequest) (*user.RegisterUserResponse, error) {
	resp := &user.RegisterUserResponse{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
	}

	if in == nil {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "request is nil"
		return resp, nil
	}

	username := strings.TrimSpace(in.Username)
	password := strings.TrimSpace(in.Password)
	if username == "" || password == "" {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "username and password are required"
		return resp, nil
	}

	// 布隆过滤器快速过滤，如果布隆过滤器没找到，说明一定没有这个用户
	if l.svcCtx.Bloom != nil {
		exists, err := l.svcCtx.Bloom.Exists([]byte(username))
		if err != nil {
			l.Logger.Errorf("register user bloom exists failed: %v", err)
		} else if exists {
			if _, err := l.svcCtx.UserModel.FindOneByUsername(l.ctx, username); err == nil {
				resp.StatusCode = errno.UserAlreadyExists
				resp.StatusMsg = "user already exists"
				return resp, nil
			}
		}
	}

    if _, err := l.svcCtx.UserModel.FindOneByUsername(l.ctx, username); err == nil {
        resp.StatusCode = errno.UserAlreadyExists
        resp.StatusMsg  = "user already exists"
        return resp, nil
    } else if err != model.ErrNotFound {
        l.Logger.Errorf("find user by username failed: %v", err)
        resp.StatusCode = errno.InternalError
        resp.StatusMsg  = "db error"
        return resp, nil
    }

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

    newId := snowflake.Next()
    if _, err := l.svcCtx.UserModel.Insert(l.ctx, &model.Users{
        Id:       uint64(newId),
        Username: username,
        Password: string(hashedPwd),
    }); err != nil {
        if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
            resp.StatusCode = errno.UserAlreadyExists
            resp.StatusMsg  = "user already exists"
            return resp, nil
        }
        l.Logger.Errorf("insert user failed: %v", err)
        resp.StatusCode = errno.InternalError
        resp.StatusMsg  = "insert user failed"
        return resp, nil
    }

    // casbin 角色
    if _, err := l.svcCtx.Casbin.AddRoleForUser(strconv.FormatInt(newId, 10), "user"); err != nil {
        l.Logger.Errorf("casbin add role failed: %v", err)
    }

    created, err := l.svcCtx.UserModel.FindOne(l.ctx, uint64(newId))
    if err != nil {
		l.Logger.Error("create user error: ", err)
        return nil, err
    }

	if l.svcCtx.Bloom != nil {
		if err := l.svcCtx.Bloom.Add([]byte(username)); err != nil {
			l.Logger.Errorf("register user bloom add failed: %v", err)
		}
	}

    resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	resp.User = userToProfile(created)

	return resp, nil
}
