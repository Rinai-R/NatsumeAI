package search

import (
	"math"
	"strings"

	"NatsumeAI/app/services/agent/agent"
	"NatsumeAI/app/services/agent/internal/agent/intent"
)

type Plan struct {
	keywords string
	minPrice int64
	maxPrice int64
	limit    int32
	slot     string
}

func (p Plan) Arguments() map[string]any {
	args := make(map[string]any)
	if p.keywords != "" {
		args["keywords"] = p.keywords
	}
	if p.minPrice > 0 {
		args["min_price"] = p.minPrice
	}
	if p.maxPrice > 0 {
		args["max_price"] = p.maxPrice
	}
	if p.limit > 0 {
		args["limit"] = p.limit
	}
	if p.slot != "" {
		args["slot"] = p.slot
	}
	return args
}

const (
	defaultLimit      int32 = 5
	outfitPlanDefault       = 3
)

func BuildPlans(req *agent.ChatReq, decision *intent.Decision) []Plan {
	queryText := ""
	if req != nil {
		queryText = strings.TrimSpace(req.GetQuery())
	}

	baseKeywords := extractKeywords(decision)
	if baseKeywords == "" {
		baseKeywords = queryText
	}

	var plans []Plan
	if decision != nil && decision.Intent == intent.IntentOutfit && len(decision.OutfitPlan) > 0 {
		for _, slot := range decision.OutfitPlan {
			plan := Plan{
				keywords: joinAndTrim(slot.Keywords),
				limit:    outfitPlanDefault,
				slot:     strings.TrimSpace(slot.Name),
				minPrice: yuanToFen(slot.BudgetMin),
				maxPrice: yuanToFen(slot.BudgetMax),
			}
			if plan.keywords == "" {
				plan.keywords = baseKeywords
			}
			if plan.limit <= 0 || plan.limit > 10 {
				plan.limit = outfitPlanDefault
			}
			addPlan(&plans, plan, queryText, decision)
		}
		return plans
	}

	basePlan := Plan{
		keywords: baseKeywords,
		limit:    defaultLimit,
	}
	if decision != nil {
		basePlan.minPrice = yuanToFen(decision.Budget.Min)
		basePlan.maxPrice = yuanToFen(decision.Budget.Max)
	}
	addPlan(&plans, basePlan, queryText, decision)

	if len(plans) == 0 && queryText != "" {
		addPlan(&plans, Plan{
			keywords: queryText,
			limit:    defaultLimit,
		}, queryText, decision)
	}

	return plans
}

func addPlan(plans *[]Plan, plan Plan, queryText string, decision *intent.Decision) {
	finalizePlan(&plan, queryText, decision)
	if plan.keywords == "" {
		return
	}

	*plans = append(*plans, plan)
}

func finalizePlan(plan *Plan, queryText string, decision *intent.Decision) {
	if plan == nil {
		return
	}
	plan.keywords = strings.TrimSpace(plan.keywords)
	if plan.keywords == "" {
		plan.keywords = strings.TrimSpace(queryText)
	}
	if plan.limit <= 0 || plan.limit > 20 {
		plan.limit = defaultLimit
	}
	if plan.slot != "" {
		plan.slot = strings.TrimSpace(plan.slot)
	}
	if plan.minPrice > 0 && plan.maxPrice > 0 && plan.minPrice > plan.maxPrice {
		plan.maxPrice = 0
	}
	if plan.minPrice == 0 && decision != nil {
		plan.minPrice = yuanToFen(decision.Budget.Min)
	}
	if plan.maxPrice == 0 && decision != nil {
		plan.maxPrice = yuanToFen(decision.Budget.Max)
	}
}

func extractKeywords(decision *intent.Decision) string {
	if decision == nil || len(decision.Keywords) == 0 {
		return ""
	}
	return joinAndTrim(decision.Keywords)
}

func joinAndTrim(words []string) string {
	clean := make([]string, 0, len(words))
	for _, w := range words {
		if trimmed := strings.TrimSpace(w); trimmed != "" {
			clean = append(clean, trimmed)
		}
	}
	return strings.Join(clean, " ")
}

func yuanToFen(v int64) int64 {
	if v <= 0 {
		return 0
	}
	if v > math.MaxInt64/100 {
		return math.MaxInt64
	}
	return v * 100
}
