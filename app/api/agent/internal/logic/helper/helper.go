package helper

import (
	"NatsumeAI/app/api/agent/internal/types"
	"NatsumeAI/app/services/agent/agentservice"
)


func ToChatResponse(res *agentservice.ChatResp) *types.ChatResponse {
	resp := &types.ChatResponse{
		Status_code: res.StatusCode,
		Status_msg: res.StatusMsg,
		Answer: res.Answer,
	}
	items := make([]types.Recommendation, 0, len(res.Items))
	for _, v := range res.Items {
		items = append(items, types.Recommendation{
			Product_id: v.ProductId,
			Name: v.Name,
			Reason: v.Reason,
			Price: v.Price,
			Picture: v.Picture,
			Categories: v.Categories,
		})
	}
	resp.Items = items
	return resp
}