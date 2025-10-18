// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"NatsumeAI/app/api/user/internal/svc"
	"NatsumeAI/app/api/user/internal/types"
	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/services/user/userservice"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/x/errors"
)

type DeleteUserAddressLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteUserAddressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteUserAddressLogic {
	return &DeleteUserAddressLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteUserAddressLogic) DeleteUserAddress(req *types.DeleteAddressRequest) error {
	userID, err := userIDFromCtx(l.ctx)
	if err != nil {
		return err
	}

	in := &userservice.DeleteAddressRequest{
		UserId:    userID,
		AddressId: req.AddressId,
	}

	res, err := l.svcCtx.UserRpc.DeleteAddress(l.ctx, in)
	if err != nil {
		l.Logger.Error("logic: delete address failed: ", err)
		return err
	}

	if res == nil {
		return errors.New(int(errno.InternalError), "empty delete address response")
	}

	if res.StatusCode != errno.StatusOK {
		l.Logger.Error("logic: delete address failed: ", res)
		return errors.New(int(res.StatusCode), res.StatusMsg)
	}

	if !res.Deleted {
		return errors.New(int(errno.InternalError), "delete address not confirmed")
	}

	return nil
}
