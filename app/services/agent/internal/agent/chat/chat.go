package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"NatsumeAI/app/services/agent/agent"
	"NatsumeAI/app/services/agent/internal/svc"
	productpb "NatsumeAI/app/services/product/product"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	defaultSearchLimit int32 = 5
	maxToolIterations        = 8
)

type Agent struct {
	log        logx.Logger
	model      *ark.ChatModel
	productCli productpb.ProductServiceClient
	intent     *intentClassifier
}

func NewAgent(ctx context.Context, svcCtx *svc.ServiceContext) *Agent {
	a := &Agent{
		log:        logx.WithContext(ctx),
		model:      svcCtx.ChatModel,
		productCli: svcCtx.ProductRpc,
	}

	if svcCtx.ChatModel != nil {
		classifier, err := newIntentClassifier(ctx, a.log, svcCtx.ChatModel)
		if err != nil {
			a.log.Errorf("init intent classifier failed: %v", err)
		} else {
			a.intent = classifier
		}
	}

	return a
}

func (a *Agent) Chat(ctx context.Context, req *agent.ChatReq) (*agent.ChatResp, error) {
	resp := &agent.ChatResp{
		StatusCode: 0,
		StatusMsg:  "ok",
	}

	if req == nil {
		resp.StatusCode = 400
		resp.StatusMsg = "invalid request"
		return resp, nil
	}

	decision, err := a.decideIntent(ctx, req)
	if err != nil {
		a.log.Errorf("intent decision failed: %v", err)
		return &agent.ChatResp{
			StatusCode: 500,
			StatusMsg:  "intent decision error",
			Answer:     "抱歉，助手暂时无法理解本次需求，请稍后再试。",
		}, nil
	}
	if decision != nil && !decision.shouldRecommend() {
		resp.Answer = a.noRecommendationAnswer(ctx, req)
		return resp, nil
	}

	if a.model == nil {
		return &agent.ChatResp{
			StatusCode: 500,
			StatusMsg:  "chat model unavailable",
			Answer:     "抱歉，推荐服务暂时不可用，请稍后再试。",
		}, nil
	}
	if a.productCli == nil {
		resp.StatusCode = 500
		resp.StatusMsg = "product rpc unavailable"
		resp.Answer = "抱歉，商品服务暂时不可用，请稍后再试。"
		return resp, nil
	}

	result, err := a.runToolDrivenChat(ctx, req, decision)
	if err != nil {
		a.log.Errorf("tool driven chat failed: %v", err)
		return &agent.ChatResp{
			StatusCode: 500,
			StatusMsg:  "agent execution error",
			Answer:     "抱歉，助手处理本次请求时出现问题，请稍后再试。",
		}, nil
	}
	return result, nil
}

func (a *Agent) decideIntent(ctx context.Context, req *agent.ChatReq) (*intentDecision, error) {
	if req == nil {
		return nil, fmt.Errorf("empty request")
	}
	if a.intent == nil {
		return nil, fmt.Errorf("intent classifier unavailable")
	}

	res, err := a.intent.analyze(ctx, intentChainInput{
		UserID: req.GetUserId(),
		Query:  req.GetQuery(),
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("empty intention result")
	}

	if res.Intent == intentOutfit && len(res.OutfitPlan) == 0 {
		res.OutfitPlan = defaultOutfitPlan()
	}
	return res, nil
}

func (a *Agent) runToolDrivenChat(ctx context.Context, req *agent.ChatReq, intent *intentDecision) (*agent.ChatResp, error) {
	toolInfos := a.buildToolInfos()
	toolModel, err := a.model.WithTools(toolInfos)
	if err != nil {
		return nil, err
	}

	systemPrompt := `你是电商导购助手。必须通过以下工具获取商品信息，不得凭空编造。
工具说明：
1. search_products：按关键词/品类检索商品。允许字段：
   - keywords string 可选
   - categories [string] 可选
   - min_price / max_price int 可选，单位为分
   - limit int 可选，默认 5
   - slot string 可选，若需搭配可注明该单品用途
2. feed_products：根据用户历史推荐商品。字段：
   - user_id int 必填
   - limit int 可选，默认 5

调用规则：
- 先理解用户意图，按需调用工具，可以多次调用 search_products 组合搭配。
- 工具返回 JSON，你需要结合返回结果为用户生成回答。
- 若用户明确表示不需要推荐，不要调用工具，直接礼貌回应。
- 最终回答必须基于工具返回的内容，语言自然、友好。`

	userPrompt := a.composeUserPrompt(req, intent)
	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}

	session := newChatSession()
	var finalMsg *schema.Message

	for i := 0; i < maxToolIterations; i++ {
		reply, err := toolModel.Generate(ctx, messages)
		if err != nil {
			return nil, err
		}
		if reply == nil {
			return nil, errors.New("model returned empty message")
		}

		messages = append(messages, reply)

		if len(reply.ToolCalls) == 0 {
			finalMsg = reply
			break
		}

		for _, call := range reply.ToolCalls {
			toolMsg, trace, recs, err := a.handleToolCall(ctx, call, req)
			if err != nil {
				a.log.Errorf("tool execution error: %v", err)
				errorContent := fmt.Sprintf(`{"error":"%s"}`, sanitizeJSONError(err))
				toolMsg = schema.ToolMessage(errorContent, enforceToolCallID(call), schema.WithToolName(call.Function.Name))
			} else {
				session.addTrace(trace)
				session.addItems(recs)
			}
			messages = append(messages, toolMsg)
		}
	}

	items := session.itemsSlice()
	sort.Slice(items, func(i, j int) bool {
		return items[i].ProductId < items[j].ProductId
	})

	var answer string
	if finalMsg != nil {
		answer = strings.TrimSpace(finalMsg.Content)
	}
	if answer == "" {
		answer = a.composeFinalAnswer(ctx, req, intent, items, session.traces)
	}
	if strings.TrimSpace(answer) == "" {
		if len(items) == 0 {
			answer = a.noResultAnswer(ctx, req, intent)
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

func (a *Agent) handleToolCall(ctx context.Context, call schema.ToolCall, req *agent.ChatReq) (*schema.Message, *toolTrace, []*agent.Recommendation, error) {
	switch call.Function.Name {
	case "search_products":
		return a.execSearchProducts(ctx, call)
	case "feed_products":
		return a.execFeedProducts(ctx, call, req)
	default:
		return nil, nil, nil, fmt.Errorf("unknown tool %q", call.Function.Name)
	}
}

func (a *Agent) execSearchProducts(ctx context.Context, call schema.ToolCall) (*schema.Message, *toolTrace, []*agent.Recommendation, error) {
	rawArgs := make(map[string]any)
	if err := json.Unmarshal([]byte(call.Function.Arguments), &rawArgs); err != nil {
		return nil, nil, nil, fmt.Errorf("parse arguments failed: %w", err)
	}

	params := parseSearchParams(rawArgs)
	if params.limit <= 0 || params.limit > 20 {
		params.limit = defaultSearchLimit
	}

	req := &productpb.SearchProductsReq{
		Keyword:    params.keywords,
		Categories: params.categories,
		MinPrice:   params.minPrice,
		MaxPrice:   params.maxPrice,
		Limit:      params.limit,
		Offset:     0,
		SortBy:     "sold",
		SortOrder:  "desc",
	}

	resp, err := a.productCli.SearchProducts(ctx, req)
	if err != nil {
		return nil, nil, nil, err
	}
	if resp.GetStatusCode() != 0 {
		return nil, nil, nil, fmt.Errorf("search error: %s", resp.GetStatusMsg())
	}

	items := make([]productResultItem, 0, len(resp.GetProducts()))
	recs := make([]*agent.Recommendation, 0, len(resp.GetProducts()))
	for _, p := range resp.GetProducts() {
		reason := buildReason(params.keywordsList(), params.categories, params.slot)
		item := productResultItem{
			ProductID:  p.GetId(),
			Name:       p.GetName(),
			Price:      p.GetPrice(),
			Picture:    p.GetPicture(),
			Categories: p.GetCategories(),
			Reason:     reason,
		}
		items = append(items, item)
		recs = append(recs, &agent.Recommendation{
			ProductId:  item.ProductID,
			Name:       item.Name,
			Reason:     item.Reason,
			Price:      item.Price,
			Picture:    item.Picture,
			Categories: item.Categories,
		})
	}

	payload := searchProductsResult{Products: items}
	contentBytes, _ := json.Marshal(payload)

	trace := &toolTrace{
		Name:   "search_products",
		Params: rawArgs,
		Result: payload,
	}

	message := schema.ToolMessage(string(contentBytes), enforceToolCallID(call), schema.WithToolName(call.Function.Name))
	return message, trace, recs, nil
}

func (a *Agent) execFeedProducts(ctx context.Context, call schema.ToolCall, req *agent.ChatReq) (*schema.Message, *toolTrace, []*agent.Recommendation, error) {
	rawArgs := make(map[string]any)
	if err := json.Unmarshal([]byte(call.Function.Arguments), &rawArgs); err != nil {
		return nil, nil, nil, fmt.Errorf("parse arguments failed: %w", err)
	}

	userID := req.GetUserId()
	if v, ok := rawArgs["user_id"]; ok {
		if parsed, ok := toInt64(v); ok {
			userID = parsed
		}
	}
	if userID == 0 {
		return nil, nil, nil, errors.New("user_id is required for feed_products")
	}

	limit := defaultSearchLimit
	if v, ok := rawArgs["limit"]; ok {
		if parsed, ok := toInt32(v); ok && parsed > 0 && parsed <= 20 {
			limit = parsed
		}
	}

	reqPb := &productpb.FeedProductsReq{
		UserId: userID,
		Limit:  limit,
	}

	resp, err := a.productCli.FeedProducts(ctx, reqPb)
	if err != nil {
		return nil, nil, nil, err
	}
	if resp.GetStatusCode() != 0 {
		return nil, nil, nil, fmt.Errorf("feed error: %s", resp.GetStatusMsg())
	}

	items := make([]productResultItem, 0, len(resp.GetProducts()))
	recs := make([]*agent.Recommendation, 0, len(resp.GetProducts()))
	for _, p := range resp.GetProducts() {
		reason := "根据你的喜好推荐"
		item := productResultItem{
			ProductID:  p.GetId(),
			Name:       p.GetName(),
			Price:      p.GetPrice(),
			Picture:    p.GetPicture(),
			Categories: p.GetCategories(),
			Reason:     reason,
		}
		items = append(items, item)
		recs = append(recs, &agent.Recommendation{
			ProductId:  item.ProductID,
			Name:       item.Name,
			Reason:     item.Reason,
			Price:      item.Price,
			Picture:    item.Picture,
			Categories: item.Categories,
		})
	}

	payload := feedProductsResult{Products: items}
	contentBytes, _ := json.Marshal(payload)

	trace := &toolTrace{
		Name:   "feed_products",
		Params: rawArgs,
		Result: payload,
	}

	message := schema.ToolMessage(string(contentBytes), enforceToolCallID(call), schema.WithToolName(call.Function.Name))
	return message, trace, recs, nil
}

func (a *Agent) composeFinalAnswer(ctx context.Context, req *agent.ChatReq, intent *intentDecision, items []*agent.Recommendation, traces []*toolTrace) string {
	if a.model == nil {
		return "抱歉，助手暂时无法生成自然语言回答。"
	}

	if len(items) == 0 {
		return a.noResultAnswer(ctx, req, intent)
	}

	var sb strings.Builder
	sb.WriteString("用户原始输入：")
	sb.WriteString(req.Query)
	sb.WriteString("\n识别意图：")
	if intent != nil {
		sb.WriteString(intent.Intent)
	} else {
		sb.WriteString(intentRecommend)
	}
	if intent != nil && len(intent.Styles) > 0 {
		sb.WriteString("\n风格偏好：")
		sb.WriteString(strings.Join(intent.Styles, "、"))
	}
	if intent != nil && (intent.Budget.Min > 0 || intent.Budget.Max > 0) {
		sb.WriteString("\n预算约束：")
		sb.WriteString(fmt.Sprintf("min=%d, max=%d", intent.Budget.Min, intent.Budget.Max))
	}
	if len(items) > 0 {
		sb.WriteString("\n候选商品：\n")
		for idx, item := range items {
			sb.WriteString(fmt.Sprintf("%d. %s | 价格: %.2f | 品类: %s | 理由: %s\n",
				idx+1, item.GetName(), float64(item.GetPrice())/100.0, strings.Join(item.GetCategories(), "/"), item.GetReason()))
		}
	}
	if len(traces) > 0 {
		sb.WriteString("\n工具调用记录：\n")
		for _, trace := range traces {
			sb.WriteString(fmt.Sprintf("- 工具: %s, 参数: %+v\n", trace.Name, trace.Params))
		}
	}

	systemPrompt := `你是一个贴心的中文电商导购顾问，需要围绕工具返回的商品信息给用户做出自然语言回答。
要求：
1. 不要杜撰商品。
2. 回答里引用商品时要用自然语言描述，可按序号列出。
3. 若是搭配需求，要说明整体风格，并串联每个单品的作用。
4. 若结果数量不足，要诚恳告知，并给出建议。
5. 答案控制在4段内，语气友好专业。`

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(sb.String()),
	}

	out, err := a.model.Generate(ctx, messages)
	if err != nil || out == nil {
		return "为你挑选了几件商品，请查看推荐列表。"
	}
	answer := strings.TrimSpace(out.Content)
	if answer == "" {
		return "为你挑选了几件商品，请查看推荐列表。"
	}
	return answer
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

func (a *Agent) noResultAnswer(ctx context.Context, req *agent.ChatReq, intent *intentDecision) string {
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

func (a *Agent) buildToolInfos() []*schema.ToolInfo {
	searchParams := map[string]*schema.ParameterInfo{
		"keywords": {
			Type: schema.String,
			Desc: "检索关键词，可为空或一句话描述",
		},
		"categories": {
			Type: schema.Array,
			Desc: "目标品类列表，可为空",
			ElemInfo: &schema.ParameterInfo{
				Type: schema.String,
			},
		},
		"min_price": {
			Type: schema.Integer,
			Desc: "最小价格，单位分",
		},
		"max_price": {
			Type: schema.Integer,
			Desc: "最大价格，单位分",
		},
		"limit": {
			Type: schema.Integer,
			Desc: "返回数量，默认 5，最大 20",
		},
		"slot": {
			Type: schema.String,
			Desc: "搭配场景下的单品用途描述",
		},
	}

	feedParams := map[string]*schema.ParameterInfo{
		"user_id": {
			Type:     schema.Integer,
			Desc:     "用户ID，必须提供",
			Required: true,
		},
		"limit": {
			Type: schema.Integer,
			Desc: "返回数量，默认 5，最大 20",
		},
	}

	return []*schema.ToolInfo{
		{
			Name:        "search_products",
			Desc:        "搜索符合条件的商品，支持关键词、品类、价格区间等筛选，可用于服饰搭配中各个单品的查找",
			ParamsOneOf: schema.NewParamsOneOfByParams(searchParams),
		},
		{
			Name:        "feed_products",
			Desc:        "根据用户历史和偏好推荐商品，用于补充或找不到结果时的候选",
			ParamsOneOf: schema.NewParamsOneOfByParams(feedParams),
		},
	}
}

func (a *Agent) composeUserPrompt(req *agent.ChatReq, intent *intentDecision) string {
	var sb strings.Builder
	sb.WriteString("用户发起对话，希望获得商品推荐。\n")
	sb.WriteString("用户输入：")
	sb.WriteString(req.Query)
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("用户ID：%d\n", req.GetUserId()))
	if intent != nil {
		sb.WriteString(fmt.Sprintf("意图推断：%s\n", intent.Intent))
		if len(intent.OutfitPlan) > 0 {
			sb.WriteString("搭配计划建议：")
			for _, slot := range intent.OutfitPlan {
				sb.WriteString(fmt.Sprintf("[%s: 类别=%s, 关键词=%s] ",
					slot.Name, strings.Join(slot.Categories, "/"), strings.Join(slot.Keywords, "/")))
			}
			sb.WriteString("\n")
		}
	}
	sb.WriteString("请根据需求合理规划工具调用，必要时多次调用 search_products 完成搭配。\n")
	return sb.String()
}

type chatSession struct {
	itemsMap map[int64]*agent.Recommendation
	traces   []*toolTrace
}

func newChatSession() *chatSession {
	return &chatSession{
		itemsMap: make(map[int64]*agent.Recommendation),
		traces:   make([]*toolTrace, 0),
	}
}

func (c *chatSession) addItems(items []*agent.Recommendation) {
	for _, item := range items {
		if item == nil || item.ProductId == 0 {
			continue
		}
		c.itemsMap[item.ProductId] = item
	}
}

func (c *chatSession) addTrace(trace *toolTrace) {
	if trace != nil {
		c.traces = append(c.traces, trace)
	}
}

func (c *chatSession) itemsSlice() []*agent.Recommendation {
	items := make([]*agent.Recommendation, 0, len(c.itemsMap))
	for _, v := range c.itemsMap {
		items = append(items, v)
	}
	return items
}

type searchParams struct {
	keywords   string
	categories []string
	minPrice   int64
	maxPrice   int64
	limit      int32
	slot       string
}

func (s searchParams) keywordsList() []string {
	if s.keywords == "" {
		return nil
	}
	parts := strings.Split(s.keywords, ",")
	results := make([]string, 0, len(parts))
	for _, part := range parts {
		trim := strings.TrimSpace(part)
		if trim != "" {
			results = append(results, trim)
		}
	}
	return results
}

func parseSearchParams(raw map[string]any) searchParams {
	params := searchParams{}

	if v, ok := raw["keywords"]; ok {
		params.keywords = joinStringSlice(v)
	}
	if v, ok := raw["categories"]; ok {
		params.categories = toStringSlice(v)
	}
	if v, ok := raw["min_price"]; ok {
		if parsed, ok := toInt64(v); ok {
			params.minPrice = parsed
		}
	}
	if v, ok := raw["max_price"]; ok {
		if parsed, ok := toInt64(v); ok {
			params.maxPrice = parsed
		}
	}
	if v, ok := raw["limit"]; ok {
		if parsed, ok := toInt32(v); ok {
			params.limit = parsed
		}
	}
	if v, ok := raw["slot"]; ok {
		params.slot = fmt.Sprint(v)
	}

	return params
}

func toStringSlice(v any) []string {
	switch val := v.(type) {
	case nil:
		return nil
	case []string:
		return val
	case []any:
		result := make([]string, 0, len(val))
		for _, item := range val {
			result = append(result, strings.TrimSpace(fmt.Sprint(item)))
		}
		return result
	default:
		str := strings.TrimSpace(fmt.Sprint(val))
		if str == "" {
			return nil
		}
		return []string{str}
	}
}

func joinStringSlice(v any) string {
	slice := toStringSlice(v)
	return strings.Join(slice, ", ")
}

func toInt64(v any) (int64, bool) {
	switch val := v.(type) {
	case float64:
		return int64(val), true
	case float32:
		return int64(val), true
	case int:
		return int64(val), true
	case int32:
		return int64(val), true
	case int64:
		return val, true
	case json.Number:
		parsed, err := val.Int64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func toInt32(v any) (int32, bool) {
	if val, ok := toInt64(v); ok {
		return int32(val), true
	}
	return 0, false
}

func buildReason(keywords []string, categories []string, slot string) string {
	parts := make([]string, 0, 3)
	if slot != "" {
		parts = append(parts, fmt.Sprintf("适合搭配中的%s", slot))
	}
	if len(keywords) > 0 {
		parts = append(parts, fmt.Sprintf("命中关键词「%s」", strings.Join(keywords, "、")))
	}
	if len(categories) > 0 {
		parts = append(parts, fmt.Sprintf("属于品类「%s」", strings.Join(categories, "、")))
	}
	if len(parts) == 0 {
		return "商品信息符合当前需求"
	}
	return strings.Join(parts, "，")
}

type productResultItem struct {
	ProductID  int64    `json:"product_id"`
	Name       string   `json:"name"`
	Price      int64    `json:"price"`
	Picture    string   `json:"picture"`
	Categories []string `json:"categories"`
	Reason     string   `json:"reason"`
}

type searchProductsResult struct {
	Products []productResultItem `json:"products"`
}

type feedProductsResult struct {
	Products []productResultItem `json:"products"`
}

type toolTrace struct {
	Name   string
	Params map[string]any
	Result any
}

func sanitizeJSONError(err error) string {
	msg := strings.ReplaceAll(err.Error(), `"`, "'")
	return strings.ReplaceAll(msg, "\n", " ")
}

func enforceToolCallID(call schema.ToolCall) string {
	if call.ID != "" {
		return call.ID
	}
	return fmt.Sprintf("call-%d", time.Now().UnixNano())
}

func summarizeRecommendations(items []*agent.Recommendation) string {
	if len(items) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("为你挑选了以下商品，请查看列表：\n")
	for idx, item := range items {
		sb.WriteString(fmt.Sprintf("%d. %s，价格约%.2f元。%s\n",
			idx+1, item.GetName(), float64(item.GetPrice())/100.0, item.GetReason()))
	}
	return strings.TrimSpace(sb.String())
}
