package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"NatsumeAI/app/common/consts/errno"
	"NatsumeAI/app/services/agent/agent"
	"NatsumeAI/app/services/agent/internal/agent/search"
	productpb "NatsumeAI/app/services/product/product"

	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
)

type Executor struct {
	log           logx.Logger
	productClient productpb.ProductServiceClient
	defaultLimit  int32
}

type productResult struct {
	ProductID  int64    `json:"product_id"`
	Name       string   `json:"name"`
	Price      int64    `json:"price"`
	Picture    string   `json:"picture"`
	Categories []string `json:"categories"`
	Reason     string   `json:"reason"`
}

type searchPayload struct {
	Products []productResult `json:"products"`
}

func NewExecutor(log logx.Logger, client productpb.ProductServiceClient, defaultLimit int32) *Executor {
	return &Executor{
		log:           log,
		productClient: client,
		defaultLimit:  defaultLimit,
	}
}

func (e *Executor) HandleSearch(ctx context.Context, call schema.ToolCall) (*schema.Message, *Trace, []*agent.Recommendation, error) {
	if e == nil || e.productClient == nil {
		return nil, nil, nil, fmt.Errorf("search executor unavailable")
	}

	rawArgs := make(map[string]any)
	if err := json.Unmarshal([]byte(call.Function.Arguments), &rawArgs); err != nil {
		return nil, nil, nil, fmt.Errorf("parse arguments failed: %w", err)
	}

	params := search.ParseParams(rawArgs)
	if params.Limit <= 0 || params.Limit > 20 {
		params.Limit = e.defaultLimit
	}

	req := &productpb.SearchProductsReq{
		Keyword:   params.Keywords,
		MinPrice:  params.MinPrice,
		MaxPrice:  params.MaxPrice,
		Limit:     params.Limit,
		Offset:    0,
		SortBy:    "sold",
		SortOrder: "desc",
	}

	resp, err := e.productClient.SearchProducts(ctx, req)
	if err != nil {
		return nil, nil, nil, err
	}
	if resp.GetStatusCode() != errno.StatusOK {
		return nil, nil, nil, fmt.Errorf("search error: %s", resp.GetStatusMsg())
	}

	items := make([]productResult, 0, len(resp.GetProducts()))
	recs := make([]*agent.Recommendation, 0, len(resp.GetProducts()))
	for _, p := range resp.GetProducts() {
		item := productResult{
			ProductID:  p.GetId(),
			Name:       p.GetName(),
			Price:      p.GetPrice(),
			Picture:    p.GetPicture(),
			Categories: p.GetCategories(),
			Reason:     "",
		}
		items = append(items, item)
		recs = append(recs, &agent.Recommendation{
			ProductId:  item.ProductID,
			Name:       item.Name,
			Price:      item.Price,
			Picture:    item.Picture,
			Categories: item.Categories,
		})
	}

	payload := searchPayload{Products: items}
	contentBytes, _ := json.Marshal(payload)

	trace := &Trace{
		Name:   "search_products",
		Params: rawArgs,
		Result: payload,
	}

	callID := call.ID
	if callID == "" {
		callID = fmt.Sprintf("call-%d", time.Now().UnixNano())
	}

	message := schema.ToolMessage(string(contentBytes), callID, schema.WithToolName(call.Function.Name))
	return message, trace, recs, nil
}
