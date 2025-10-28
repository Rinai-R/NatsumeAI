package logic

import (
	"context"
	"fmt"

	"NatsumeAI/app/services/agent/agent"
	reviewer "NatsumeAI/app/services/agent/internal/agent/review"
	"NatsumeAI/app/services/agent/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReviewMerchantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReviewMerchantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReviewMerchantLogic {
    return &ReviewMerchantLogic{
        ctx:    ctx,
        svcCtx: svcCtx,
        Logger: logx.WithContext(ctx),
    }
}

func (l *ReviewMerchantLogic) ReviewMerchant(in *agent.ReviewMerchantReq) (*agent.ReviewMerchantResult, error) {

    app := in.GetApplication()

    summary := fmt.Sprintf("申请ID:%d\n用户ID:%d\n店铺名称:%s\n联系人:%s\n电话:%s\n地址:%s\n申请理由:%s",
        in.GetApplicationId(), in.GetUserId(),
        safeStr(app.GetShopName()), safeStr(app.GetContactName()), safeStr(app.GetContactPhone()), safeStr(app.GetAddress()), safeStr(app.GetDescription()))


    out := reviewer.NewReviewer(l.ctx, l.svcCtx).Review(summary)

    res := &agent.ReviewMerchantResult{
        ApplicationId: in.GetApplicationId(),
    }
    if out == nil {
        res.Decision = agent.ReviewDecision_REVIEW_DECISION_UNKNOWN
        res.Reason = "系统繁忙或模型不可用"
        fmt.Println(res)
        return res, nil
    }

    if out.Ok {
        res.Decision = agent.ReviewDecision_REVIEW_DECISION_APPROVE
    } else {
        res.Decision = agent.ReviewDecision_REVIEW_DECISION_REJECT
    }
    res.Reason = out.Reason
    fmt.Println(res)
    return res, nil
}

func safeStr(s string) string {
    if s == "" {
        return "-"
    }
    return s
}
