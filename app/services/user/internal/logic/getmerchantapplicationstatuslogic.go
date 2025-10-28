package logic

import (
    "context"

    "NatsumeAI/app/services/user/internal/svc"
    "NatsumeAI/app/services/user/user"

    "github.com/zeromicro/go-zero/core/logx"
)

type GetMerchantApplicationStatusLogic struct {
    ctx    context.Context
    svcCtx *svc.ServiceContext
    logx.Logger
}

func NewGetMerchantApplicationStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMerchantApplicationStatusLogic {
    return &GetMerchantApplicationStatusLogic{
        ctx:    ctx,
        svcCtx: svcCtx,
        Logger: logx.WithContext(ctx),
    }
}

func (l *GetMerchantApplicationStatusLogic) GetMerchantApplicationStatus(in *user.GetMerchantApplicationStatusRequest) (*user.GetMerchantApplicationStatusResponse, error) {
    resp := &user.GetMerchantApplicationStatusResponse{StatusCode: 0, StatusMsg: "OK"}
    if in.GetApplicationId() <= 0 {
        resp.StatusCode = -1
        resp.StatusMsg = "invalid application_id"
        return resp, nil
    }
    row, err := l.svcCtx.MerchantsModel.FindOne(l.ctx, in.GetApplicationId())
    if err != nil || row == nil {
        resp.StatusCode = -2
        resp.StatusMsg = "not found"
        return resp, nil
    }
    resp.ApplicationId = row.Id
    resp.ApplicationStatus = row.Status
    resp.RejectReason = row.RejectReason
    if row.ReviewedAt.Valid {
        resp.ReviewedAt = row.ReviewedAt.Time.Unix()
    }
    return resp, nil
}
