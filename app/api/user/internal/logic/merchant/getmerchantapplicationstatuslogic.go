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

type GetMerchantApplicationStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMerchantApplicationStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMerchantApplicationStatusLogic {
	return &GetMerchantApplicationStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMerchantApplicationStatusLogic) GetMerchantApplicationStatus(applicationId int64) (resp *types.GetMerchantApplicationStatusResponse, err error) {
    uid, err := util.UserIdFromCtx(l.ctx)
    if err != nil {
        return nil, err
    }
    in := &userservice.GetMerchantApplicationStatusRequest{UserId: uid, ApplicationId: applicationId}
    r, err := l.svcCtx.UserRpc.GetMerchantApplicationStatus(l.ctx, in)
    if err != nil {
        return nil, err
    }
    return &types.GetMerchantApplicationStatusResponse{
        ApplicationId:     r.GetApplicationId(),
        ApplicationStatus: r.GetApplicationStatus(),
        RejectReason:      r.GetRejectReason(),
        ReviewedAt:        r.GetReviewedAt(),
    }, nil
}
