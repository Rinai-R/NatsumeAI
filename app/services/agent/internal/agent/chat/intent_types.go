package chat

const (
	intentRecommend = "recommend"
	intentOutfit    = "outfit"
	intentNoNeed    = "no_need"
)

type budgetInfo struct {
	Min int64 `json:"min,omitempty"`
	Max int64 `json:"max,omitempty"`
}

type outfitSlot struct {
	Name       string   `json:"name,omitempty"`
	Categories []string `json:"categories,omitempty"`
	Keywords   []string `json:"keywords,omitempty"`
	BudgetMin  int64    `json:"budget_min,omitempty"`
	BudgetMax  int64    `json:"budget_max,omitempty"`
}

type intentDecision struct {
	Intent        string       `json:"intent"`
	Keywords      []string     `json:"keywords,omitempty"`
	Categories    []string     `json:"categories,omitempty"`
	Styles        []string     `json:"styles,omitempty"`
	Budget        budgetInfo   `json:"budget"`
	OutfitPlan    []outfitSlot `json:"outfit_plan,omitempty"`
	UserRejection bool         `json:"user_rejection,omitempty"`
	Explanation   string       `json:"explanation,omitempty"`
	RawOutput     string       `json:"-"`
}

func (d *intentDecision) shouldRecommend() bool {
	if d == nil {
		return true
	}
	if d.UserRejection {
		return false
	}
	return d.Intent != intentNoNeed
}

func defaultOutfitPlan() []outfitSlot {
	return []outfitSlot{
		{Name: "上衣", Categories: []string{"tops", "shirt"}, Keywords: []string{"搭配", "上衣"}},
		{Name: "下装", Categories: []string{"pants", "skirt"}, Keywords: []string{"搭配", "下装"}},
		{Name: "鞋履", Categories: []string{"shoes"}, Keywords: []string{"鞋"}},
	}
}
