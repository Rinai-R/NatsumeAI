package chat

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

type intentChainInput struct {
	UserID int64
	Query  string
}

type intentClassifier struct {
	log      logx.Logger
	runnable compose.Runnable[intentChainInput, *intentDecision]
}

func newIntentClassifier(ctx context.Context, logger logx.Logger, chatModel model.BaseChatModel) (*intentClassifier, error) {
	if chatModel == nil {
		return nil, fmt.Errorf("chat model is required")
	}

	chain := compose.NewChain[intentChainInput, *intentDecision]()

	chain.AppendLambda(compose.InvokableLambda(func(_ context.Context, in intentChainInput) ([]*schema.Message, error) {
		systemPrompt := `你是电商导购的意图分析助手。请阅读用户输入，推断用户的购物意图，并输出JSON，不要解释，也不要添加额外文本。JSON字段说明：
{
  "intent": "recommend|no_need|outfit",
  "keywords": ["提取的核心关键词"],
  "categories": ["建议关联的品类，英文或常用品类名"],
  "styles": ["风格关键词，可为空"],
  "budget": {"min": 数字元, "max": 数字元},
  "outfit_plan": [
    {"name": "单品名称，如上衣", "categories": ["建议类别"], "keywords": ["追加关键词"], "budget_min": 数字, "budget_max": 数字}
  ],
  "user_rejection": true|false,
  "explanation": "简要理由"
}
若无法确定预算或搭配信息，请保持字段为0或空数组。请务必返回合法JSON，不要包含多余字符。`

		var user strings.Builder
		user.WriteString("用户原始输入：")
		user.WriteString(in.Query)
		user.WriteString("\n用户ID：")
		user.WriteString(fmt.Sprintf("%d", in.UserID))

		return []*schema.Message{
			schema.SystemMessage(systemPrompt),
			schema.UserMessage(user.String()),
		}, nil
	}))

	chain.AppendChatModel(chatModel)

	chain.AppendLambda(compose.InvokableLambda(func(_ context.Context, msg *schema.Message) (*intentDecision, error) {
		if msg == nil {
			return nil, fmt.Errorf("empty message")
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			return nil, fmt.Errorf("empty response")
		}

		clean := trimJSONBlock(content)

		var decision intentDecision
		if err := json.Unmarshal([]byte(clean), &decision); err != nil {
			return nil, err
		}
		decision.RawOutput = clean
		if decision.Intent == "" {
			decision.Intent = intentRecommend
		}
		return &decision, nil
	}))

	runnable, err := chain.Compile(ctx)
	if err != nil {
		return nil, err
	}

	return &intentClassifier{
		log:      logger,
		runnable: runnable,
	}, nil
}

func (c *intentClassifier) analyze(ctx context.Context, in intentChainInput) (*intentDecision, error) {
	if c == nil || c.runnable == nil {
		return nil, fmt.Errorf("intent classifier unavailable")
	}
	return c.runnable.Invoke(ctx, in)
}

func trimJSONBlock(content string) string {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 || start > end {
		return content
	}
	return content[start : end+1]
}
