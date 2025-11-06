package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"NatsumeAI/app/services/agent/agent"
	"NatsumeAI/app/services/agent/internal/agent/intent"
	"NatsumeAI/app/services/agent/internal/agent/tools"

	"github.com/cloudwego/eino/schema"
)

func (a *Agent) composeFinalAnswer(ctx context.Context, req *agent.ChatReq, intentDecision *intent.Decision, items []*agent.Recommendation, traces []*tools.Trace) string {
	if a.model == nil {
		a.log.Errorf("compose final answer failed: chat model unavailable")
		return ""
	}

	if len(items) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("用户原始输入：")
	sb.WriteString(req.Query)
	sb.WriteString("\n识别意图：")
	if intentDecision != nil {
		sb.WriteString(intentDecision.Intent)
	} else {
		sb.WriteString(intent.IntentRecommend)
	}
	if intentDecision != nil && len(intentDecision.Styles) > 0 {
		sb.WriteString("\n风格偏好：")
		sb.WriteString(strings.Join(intentDecision.Styles, "、"))
	}
	if intentDecision != nil && (intentDecision.Budget.Min > 0 || intentDecision.Budget.Max > 0) {
		sb.WriteString("\n预算约束：")
		sb.WriteString(fmt.Sprintf("min=%d, max=%d", intentDecision.Budget.Min, intentDecision.Budget.Max))
	}
	if len(items) > 0 {
		sb.WriteString("\n候选商品：\n")
		for idx, item := range items {
			sb.WriteString(fmt.Sprintf("%d. %s | 价格: %.2f",
				idx+1, item.GetName(), float64(item.GetPrice())/100.0))
			if cats := strings.Join(item.GetCategories(), "/"); cats != "" {
				sb.WriteString(fmt.Sprintf(" | 品类: %s", cats))
			}
			if reason := strings.TrimSpace(item.GetReason()); reason != "" {
				sb.WriteString(fmt.Sprintf(" | 理由: %s", reason))
			}
			sb.WriteString("\n")
		}
	}
	systemPrompt := `你是中文电商导购顾问，请基于提供的工具结果输出 1-4 段推荐。
须知：
- 不得杜撰商品或信息。
- 引用商品可用序号／自然语言描述，点到名称、价格亮点、用途。
- 搭配需求需交代整体风格与各单品角色。
- 结果不足时要坦诚说明并给出改进建议。
- 语气友好专业。`

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(sb.String()),
	}

	start := time.Now()
	out, err := a.model.Generate(ctx, messages)
	a.log.Infof("compose final answer took %s", time.Since(start))
	if err != nil || out == nil {
		a.log.Errorf("compose final answer generate failed: %v", err)
		return ""
	}
	return strings.TrimSpace(out.Content)
}

func (a *Agent) noRecommendationAnswer(ctx context.Context, req *agent.ChatReq) string {
	if a.model == nil {
		return "好的，已经记下你的想法，如果之后想了解商品或搭配，随时叫我。"
	}
	prompt := `用户明确表示不需要商品推荐。请用中文简短回复，表达理解与随时待命，不附加任何商品信息。`
	msgs := []*schema.Message{
		schema.SystemMessage(prompt),
		schema.UserMessage(req.Query),
	}
	out, err := a.model.Generate(ctx, msgs)
	if err != nil || out == nil || strings.TrimSpace(out.Content) == "" {
		return "好的，已经记下你的想法，如果之后想了解商品或搭配，随时叫我。"
	}
	return strings.TrimSpace(out.Content)
}
func (a *Agent) noResultAnswer(ctx context.Context, req *agent.ChatReq, intentDecision *intent.Decision) string {
	if a.model == nil {
		return "抱歉，暂时没有找到合适的商品，可以试着调整一下需求或告诉我更多偏好。"
	}
	systemPrompt := `你是电商导购助手。本次未检索到合适的商品，请用中文回复用户，说明当前没有匹配的商品，并给出1-2条可调整的建议。`
	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(req.Query),
	}
	out, err := a.model.Generate(ctx, messages)
	if err != nil || out == nil {
		return "抱歉，暂时没有找到合适的商品，可以试着调整一下需求或告诉我更多偏好。"
	}
	answer := strings.TrimSpace(out.Content)
	if answer == "" {
		return "抱歉，暂时没有找到合适的商品，可以试着调整一下需求或告诉我更多偏好。"
	}
	return answer
}

func limitRecommendations(items []*agent.Recommendation, limit int) []*agent.Recommendation {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	result := make([]*agent.Recommendation, 0, limit)
	for i := 0; i < len(items) && len(result) < limit; i++ {
		if items[i] != nil {
			result = append(result, items[i])
		}
	}
	return result
}

func summarizeRecommendations(items []*agent.Recommendation) string {
	if len(items) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("为你挑选了以下商品，请查看列表：\n")
	for idx, item := range items {
		line := fmt.Sprintf("%d. %s，价格约%.2f元", idx+1, item.GetName(), float64(item.GetPrice())/100.0)
		if reason := strings.TrimSpace(item.GetReason()); reason != "" {
			line = fmt.Sprintf("%s。%s", line, reason)
		}
		sb.WriteString(line + "\n")
	}
	return strings.TrimSpace(sb.String())
}
