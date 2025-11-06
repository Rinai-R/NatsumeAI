package intent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	classifierModelNodeKey = "intent_classifier_model"
	classifierToolName     = "submit_intent_decision"
)

type Classifier struct {
	log      logx.Logger
	runnable compose.Runnable[Input, *Decision]
	tools    []*schema.ToolInfo
}

type Input struct {
	UserID int64
	Query  string
}

func NewClassifier(ctx context.Context, logger logx.Logger, chatModel model.BaseChatModel) (*Classifier, error) {
	if chatModel == nil {
		return nil, fmt.Errorf("chat model is required")
	}

	intentTool := buildDecisionTool()
	tools := []*schema.ToolInfo{intentTool}

	intentModel := chatModel
	if toolCapable, ok := chatModel.(model.ToolCallingChatModel); ok {
		if modelWithTools, err := toolCapable.WithTools(tools); err != nil {
			logger.Errorf("bind intent tool failed: %v", err)
		} else {
			intentModel = modelWithTools
		}
	}

	chain := compose.NewChain[Input, *Decision]()

	chain.AppendLambda(compose.InvokableLambda(func(_ context.Context, in Input) ([]*schema.Message, error) {
		systemPrompt := `你是电商导购的意图分析助手。请阅读用户输入，推断用户的购物意图，并输出JSON，不要解释，也不要添加额外文本。JSON字段说明：
{
  "intent": "recommend|no_need|outfit",
  "keywords": ["提取的核心关键词"],
  "styles": ["风格关键词，可为空"],
  "budget": {"min": 数字元, "max": 数字元},
  "outfit_plan": [
    {"name": "单品名称，如上衣", "keywords": ["追加关键词"], "budget_min": 数字, "budget_max": 数字}
  ],
  "user_rejection": true|false,
  "explanation": "简要理由"
}
若无法确定预算或搭配信息，请保持字段为0或空数组。请务必返回合法JSON，不要包含多余字符。`

		var instructions strings.Builder
		instructions.WriteString(systemPrompt)
		instructions.WriteString("\n请通过调用工具 submit_intent_decision 提交结果，参数需与上述字段一致，不要输出额外文本。")

		var user strings.Builder
		user.WriteString("用户原始输入：")
		user.WriteString(in.Query)
		user.WriteString("\n用户ID：")
		user.WriteString(fmt.Sprintf("%d", in.UserID))

		return []*schema.Message{
			schema.SystemMessage(instructions.String()),
			schema.UserMessage(user.String()),
		}, nil
	}))

	chain.AppendChatModel(intentModel, compose.WithNodeKey(classifierModelNodeKey))

	chain.AppendLambda(compose.InvokableLambda(func(_ context.Context, msg *schema.Message) (*Decision, error) {
		if msg == nil {
			return nil, fmt.Errorf("empty message")
		}

		payload := extractToolArguments(msg)
		if payload == "" {
			return nil, fmt.Errorf("intent decision tool payload missing")
		}

		var decision Decision
		if err := json.Unmarshal([]byte(payload), &decision); err != nil {
			return nil, fmt.Errorf("unmarshal intent decision: %w", err)
		}
		decision.RawOutput = payload
		if decision.Intent == "" {
			decision.Intent = IntentRecommend
		}
		return &decision, nil
	}))

	runnable, err := chain.Compile(ctx)
	if err != nil {
		return nil, err
	}

	return &Classifier{
		log:      logger,
		runnable: runnable,
		tools:    tools,
	}, nil
}

func (c *Classifier) Analyze(ctx context.Context, in Input) (*Decision, error) {
	if c == nil || c.runnable == nil {
		return nil, fmt.Errorf("intent classifier unavailable")
	}

	var opts []compose.Option
	if len(c.tools) > 0 {
		opt := compose.WithChatModelOption(
			model.WithTools(c.tools),
			model.WithToolChoice(schema.ToolChoiceForced),
		).DesignateNode(classifierModelNodeKey)
		opts = append(opts, opt)
	}

	return c.runnable.Invoke(ctx, in, opts...)
}

func extractToolArguments(msg *schema.Message) string {
	for _, call := range msg.ToolCalls {
		if strings.EqualFold(call.Function.Name, classifierToolName) {
			return strings.TrimSpace(call.Function.Arguments)
		}
	}
	return ""
}

func buildDecisionTool() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: classifierToolName,
		Desc: "提交电商导购意图分析结果的工具",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"intent": {
				Type:     schema.String,
				Desc:     "用户意图，recommend=推荐，no_need=无需推荐，outfit=搭配规划",
				Enum:     []string{IntentRecommend, IntentNoNeed, IntentOutfit},
				Required: true,
			},
			"keywords": {
				Type: schema.Array,
				Desc: "提取的核心关键词",
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
				},
			},
			"styles": {
				Type: schema.Array,
				Desc: "风格关键词，可为空",
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
				},
			},
			"budget": {
				Type: schema.Object,
				Desc: "预算范围，单位元，未知时填0",
				SubParams: map[string]*schema.ParameterInfo{
					"min": {
						Type: schema.Integer,
						Desc: "预算下限，单位元",
					},
					"max": {
						Type: schema.Integer,
						Desc: "预算上限，单位元",
					},
				},
			},
			"outfit_plan": {
				Type: schema.Array,
				Desc: "搭配规划列表",
				ElemInfo: &schema.ParameterInfo{
					Type: schema.Object,
					SubParams: map[string]*schema.ParameterInfo{
						"name": {
							Type:     schema.String,
							Desc:     "单品名称，如上衣",
							Required: true,
						},
						"keywords": {
							Type:     schema.Array,
							Desc:     "追加关键词",
							ElemInfo: &schema.ParameterInfo{Type: schema.String},
						},
						"budget_min": {
							Type: schema.Integer,
							Desc: "预算下限，单位元",
						},
						"budget_max": {
							Type: schema.Integer,
							Desc: "预算上限，单位元",
						},
					},
				},
			},
			"user_rejection": {
				Type: schema.Boolean,
				Desc: "用户是否拒绝推荐",
			},
			"explanation": {
				Type: schema.String,
				Desc: "简要理由",
			},
		}),
	}
}
