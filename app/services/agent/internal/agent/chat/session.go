package chat

import (
	"NatsumeAI/app/services/agent/agent"
	"NatsumeAI/app/services/agent/internal/agent/tools"
)

type chatSession struct {
	itemsMap map[int64]*agent.Recommendation
	traces   []*tools.Trace
}

func newChatSession() *chatSession {
	return &chatSession{
		itemsMap: make(map[int64]*agent.Recommendation),
		traces:   make([]*tools.Trace, 0),
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

func (c *chatSession) addTrace(trace *tools.Trace) {
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

func (c *chatSession) count() int {
	if c == nil {
		return 0
	}
	return len(c.itemsMap)
}
