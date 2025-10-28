// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package merchant

import (
    "context"

    "NatsumeAI/app/api/user/internal/svc"
    "NatsumeAI/app/api/user/internal/types"
    "NatsumeAI/app/common/util"
    "NatsumeAI/app/services/user/userservice"

    "github.com/zeromicro/go-zero/core/logx"
)

type ApplyMerchantLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewApplyMerchantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApplyMerchantLogic {
	return &ApplyMerchantLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ApplyMerchantLogic) ApplyMerchant(req *types.ApplyMerchantRequest) (resp *types.ApplyMerchantResponse, err error) {
    uid, err := util.UserIdFromCtx(l.ctx)
    if err != nil {
        return nil, err
    }
    in := &userservice.ApplyMerchantRequest{
        UserId: uid,
        Application: &userservice.MerchantApplicationInput{
            ShopName:     req.Application.ShopName,
            ContactName:  req.Application.ContactName,
            ContactPhone: req.Application.ContactPhone,
            Address:      req.Application.Address,
            Description:  req.Application.Description,
        },
    }
    r, err := l.svcCtx.UserRpc.ApplyMerchant(l.ctx, in)
    if err != nil {
        return nil, err
    }
    return &types.ApplyMerchantResponse{
        ApplicationId:     r.GetApplicationId(),
        ApplicationStatus: r.GetApplicationStatus(),
    }, nil
}
