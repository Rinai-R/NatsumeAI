package logic

import (
	"context"
	"strings"

	"NatsumeAI/app/common/consts/errno"
	model "NatsumeAI/app/dal/user"
	"NatsumeAI/app/services/user/internal/svc"
	"NatsumeAI/app/services/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateAddressLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateAddressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAddressLogic {
	return &CreateAddressLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateAddressLogic) CreateAddress(in *user.CreateAddressRequest) (*user.CreateAddressResponse, error) {
	resp := &user.CreateAddressResponse{
		StatusCode: errno.InternalError,
		StatusMsg:  "internal error",
	}

	if in == nil || in.Address == nil {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "address payload is required"
		return resp, nil
	}

	if in.UserId <= 0 {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "invalid user id"
		return resp, nil
	}

	detail := strings.TrimSpace(in.Address.Detail)
	if detail == "" {
		resp.StatusCode = errno.InvalidParam
		resp.StatusMsg = "address detail is required"
		return resp, nil
	}

	if _, err := l.svcCtx.UserModel.FindOne(l.ctx, uint64(in.UserId)); err != nil {
		if err == model.ErrNotFound {
			resp.StatusCode = errno.UserNotFound
			resp.StatusMsg = "user not found"
			return resp, nil
		}
		return nil, err
	}

	addr := &model.UserAddresses{
		UserId:    uint64(in.UserId),
		Detail:    detail,
		IsDefault: boolToInt64(in.Address.IsDefault),
	}

	result, err := l.svcCtx.UserAddressModel.Insert(l.ctx, addr)
	if err != nil {
		return nil, err
	}

	newID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	created, err := l.svcCtx.UserAddressModel.FindOne(l.ctx, uint64(newID))
	if err != nil {
		return nil, err
	}

	if in.Address.IsDefault {
		if err := ensureDefaultForUser(l.ctx, l.svcCtx.UserAddressModel, uint64(in.UserId), created.Id); err != nil {
			return nil, err
		}
		created.IsDefault = 1
	}

	resp.StatusCode = errno.StatusOK
	resp.StatusMsg = "ok"
	resp.Address = addressToProto(created)

	return resp, nil
}
