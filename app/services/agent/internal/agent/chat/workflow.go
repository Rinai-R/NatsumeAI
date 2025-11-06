package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"NatsumeAI/app/services/agent/agent"
	"NatsumeAI/app/services/agent/internal/agent/intent"
	"NatsumeAI/app/services/agent/internal/agent/tools"

	"github.com/cloudwego/eino/schema"
)

func (a *Agent) runToolDrivenChat(ctx context.Context, req *agent.ChatReq, decision *intent.Decision) (*agent.ChatResp, error) {
	start := time.Now()
	defer func() {
		a.log.Infof("tool driven chat total duration: %s", time.Since(start))
	}()

	toolModel, err := a.ensureToolModel()
	if err != nil {
		return nil, err
	}

	systemPrompt := `你是电商导购助手，只能依据工具返回的数据回答，禁止编造。
工具：
- search_products：按关键词、品类、价格筛选；可用参数 keywords、categories、min_price、max_price、limit(<=20)、slot。
规则：
1. 先判断意图，如需工具请在同一轮内规划并一次性发起所有必要调用，可包含多个不同参数的调用。
2. 同一类型工具最多调用3次，如已有足够数据请立即总结回复。
3. 不得重复调用参数基本相同的工具；若结果不足，优先调整参数后再尝试。
4. 回复必须来自工具 JSON，可适度总结，但不可新增虚构信息。
5. 用户明确拒绝推荐时，直接礼貌回应并跳过工具。
6. 最终回答需语言自然、突出商品亮点/搭配建议，说明所有信息来源于工具结果。`

	userPrompt := a.composeUserPrompt(req, decision)
	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}

	session := newChatSession()
	var finalMsg *schema.Message
	toolInvoked := false
	toolCalls := 0
	callSignatures := make(map[string]struct{})

	for i := 0; i < maxToolIterations; i++ {
		iterStart := time.Now()
		reply, err := toolModel.Generate(ctx, messages)
		genDuration := time.Since(iterStart)
		a.log.Infof("tool iteration %d generate took %s", i+1, genDuration)
		if err != nil {
			return nil, err
		}
		if reply == nil {
			return nil, errors.New("model returned empty message")
		}

		messages = append(messages, reply)

		if len(reply.ToolCalls) == 0 {
			if !toolInvoked {
				return nil, fmt.Errorf("model returned answer without tool usage")
			}
			finalMsg = reply
			break
		}

		toolInvoked = true
		shortCircuit := false
		for _, call := range reply.ToolCalls {
			callID := call.ID
			if callID == "" {
				callID = fmt.Sprintf("call-%d", time.Now().UnixNano())
			}

			signature := fmt.Sprintf("%s|%s", strings.ToLower(strings.TrimSpace(call.Function.Name)), strings.TrimSpace(call.Function.Arguments))
			if _, seen := callSignatures[signature]; seen {
				a.log.Infof("duplicate tool call skipped: %s", signature)
				dupPayload, _ := json.Marshal(map[string]any{
					"error": "duplicate tool invocation rejected to reduce latency",
				})
				toolMsg := schema.ToolMessage(string(dupPayload), callID, schema.WithToolName(call.Function.Name))
				messages = append(messages, toolMsg)
				continue
			}
			if toolCalls >= maxToolCallsPerSession {
				a.log.Infof("tool call limit reached (%d), skipping %s", maxToolCallsPerSession, call.Function.Name)
				limitPayload, _ := json.Marshal(map[string]any{
					"error": fmt.Sprintf("tool call limit reached (%d)", maxToolCallsPerSession),
				})
				toolMsg := schema.ToolMessage(string(limitPayload), callID, schema.WithToolName(call.Function.Name))
				messages = append(messages, toolMsg)
				shortCircuit = true
				break
			}

			callSignatures[signature] = struct{}{}
			toolCalls++
			callStart := time.Now()
			toolMsg, trace, recs, err := a.handleToolCall(ctx, call, req)
			callDuration := time.Since(callStart)
			if err != nil {
				a.log.Errorf("tool execution error: %v", err)
				errorPayload, _ := json.Marshal(map[string]string{"error": err.Error()})
				toolMsg = schema.ToolMessage(string(errorPayload), callID, schema.WithToolName(call.Function.Name))
				a.log.Infof("tool call %s failed after %s", call.Function.Name, callDuration)
			} else {
				session.addTrace(trace)
				session.addItems(recs)
				a.log.Infof("tool call %s succeeded in %s", call.Function.Name, callDuration)
			}
			messages = append(messages, toolMsg)
		}

		if shortCircuit {
			a.log.Infof("tool call budget exhausted (%d), short-circuiting to compose answer", maxToolCallsPerSession)
			break
		}
	}

	if !toolInvoked {
		return nil, fmt.Errorf("model failed to invoke any tool")
	}

	items := session.itemsSlice()
	sort.Slice(items, func(i, j int) bool {
		return items[i].ProductId < items[j].ProductId
	})
	items = limitRecommendations(items, int(defaultSearchLimit))

	var answer string
	if finalMsg != nil {
		answer = strings.TrimSpace(finalMsg.Content)
	}
	if answer == "" {
		answer = a.composeFinalAnswer(ctx, req, decision, items, session.traces)
	}
	if strings.TrimSpace(answer) == "" {
		if len(items) == 0 {
			answer = a.noResultAnswer(ctx, req, decision)
		} else {
			answer = summarizeRecommendations(items)
		}
	}

	return &agent.ChatResp{
		StatusCode: 0,
		StatusMsg:  "ok",
		Answer:     answer,
		Items:      items,
	}, nil
}

func (a *Agent) composeUserPrompt(req *agent.ChatReq, decision *intent.Decision) string {
	var sb strings.Builder
	sb.WriteString("用户请求商品推荐。\n输入：")
	sb.WriteString(req.Query)
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("用户ID：%d\n", req.GetUserId()))
	if decision != nil {
		sb.WriteString(fmt.Sprintf("意图：%s\n", decision.Intent))
		if len(decision.OutfitPlan) > 0 {
			sb.WriteString("搭配计划：")
			for _, slot := range decision.OutfitPlan {
				sb.WriteString(fmt.Sprintf("[%s | 关键词=%s] ",
					slot.Name, strings.Join(slot.Keywords, "/")))
			}
			sb.WriteString("\n")
		}
	}
	sb.WriteString("请按需规划工具调用，必要时多次 search_products 完成搭配。\n")
	return sb.String()
}

func (a *Agent) handleToolCall(ctx context.Context, call schema.ToolCall, req *agent.ChatReq) (*schema.Message, *tools.Trace, []*agent.Recommendation, error) {
	switch call.Function.Name {
	case "search_products":
		if a.toolExecutor == nil {
			return nil, nil, nil, fmt.Errorf("search executor unavailable")
		}
		return a.toolExecutor.HandleSearch(ctx, call)
	default:
		return nil, nil, nil, fmt.Errorf("unknown tool %q", call.Function.Name)
	}
}
