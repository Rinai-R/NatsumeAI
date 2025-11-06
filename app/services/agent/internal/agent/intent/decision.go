package intent

const (
	IntentRecommend = "recommend"
	IntentOutfit    = "outfit"
	IntentNoNeed    = "no_need"
)

type BudgetInfo struct {
	Min int64 `json:"min,omitempty"`
	Max int64 `json:"max,omitempty"`
}

type OutfitSlot struct {
	Name      string   `json:"name,omitempty"`
	Keywords  []string `json:"keywords,omitempty"`
	BudgetMin int64    `json:"budget_min,omitempty"`
	BudgetMax int64    `json:"budget_max,omitempty"`
}

type Decision struct {
	Intent        string       `json:"intent"`
	Keywords      []string     `json:"keywords,omitempty"`
	Styles        []string     `json:"styles,omitempty"`
	Budget        BudgetInfo   `json:"budget"`
	OutfitPlan    []OutfitSlot `json:"outfit_plan,omitempty"`
	UserRejection bool         `json:"user_rejection,omitempty"`
	Explanation   string       `json:"explanation,omitempty"`
	RawOutput     string       `json:"-"`
}

func (d *Decision) ShouldRecommend() bool {
	if d == nil {
		return true
	}
	if d.UserRejection {
		return false
	}
	return d.Intent != IntentNoNeed
}
