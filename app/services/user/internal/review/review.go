package review

import (
	"NatsumeAI/app/services/user/internal/svc"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/ark"
	mdl "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type Reviewer struct {
    ctx       context.Context
    chatModel *ark.ChatModel
}

type ReviewResp struct {
    Ok     bool
    Reason string
}

func NewReviewer(ctx context.Context, svcCtx *svc.ServiceContext) *Reviewer {
    return &Reviewer{
        ctx:       ctx,
        chatModel: svcCtx.ChatModel,
    }
}

func (r *Reviewer) Review(query string) *ReviewResp {

    if resp := r.reviewWithLLM(query); resp != nil {
        return resp
    }
	return &ReviewResp{
		Ok: false,
		Reason: "审核出现错误，请重试",
	}
}

func (r *Reviewer) reviewWithLLM(query string) *ReviewResp {

	// 强制使用 tool
    tool := &schema.ToolInfo{
        Name: "submit_decision",
        Desc: "向平台返回最终审核结论",
        ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
            "ok": {
                Type:     schema.Boolean,
                Desc:     "是否通过。true=通过，false=拒绝",
                Required: true,
            },
            "reason": {
                Type:     schema.String,
                Desc:     "简短中文原因，若通过则写明'未发现明显合规风险'或具体说明；若拒绝写明风险点",
                Required: true,
            },
        }),
    }

    sys := schema.SystemMessage(
        "你是平台合规审核助手。\n" +
            "请严格依据平台规则判断并调用工具 submit_decision 提交最终结论。\n" +
            "规则举例（非穷尽）：涉黄/赌/毒/枪支弹药/违法违规/诈骗/代开发票/假证 等必须拒绝；信息严重缺失无法判断亦拒绝；否则通过。\n" +
            "必须仅调用一次工具并给出明确 reason。",
    )
    user := schema.UserMessage("待审核申请如下：\n" + strings.TrimSpace(query))

    msg, err := r.chatModel.Generate(
        context.Background(),
        []*schema.Message{sys, user},
        mdl.WithTools([]*schema.ToolInfo{tool}),
        mdl.WithToolChoice(schema.ToolChoiceForced),
    )
    if err != nil || msg == nil {
		fmt.Println("出现错误", err)
        return nil
    }

    // 用 tool 的结果
    for _, tc := range msg.ToolCalls {
        if strings.EqualFold(tc.Function.Name, "submit_decision") {
            var args struct {
                OK     bool   `json:"ok"`
                Reason string `json:"reason"`
            }
            if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				fmt.Println("1出现错误", err)
                return nil
            }
            reason := strings.TrimSpace(args.Reason)
            if reason == "" {
                if args.OK {
                    reason = "通过：未发现明显合规风险"
                } else {
                    reason = "拒绝：模型判定存在合规风险"
                }
            }
            return &ReviewResp{Ok: args.OK, Reason: reason}
        }
    }
	fmt.Println("没有工具")
    return nil
}