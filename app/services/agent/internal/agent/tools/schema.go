package tools

import "github.com/cloudwego/eino/schema"

func BuildToolInfos() []*schema.ToolInfo {
	searchParams := map[string]*schema.ParameterInfo{
		"keywords": {
			Type: schema.String,
			Desc: "检索关键词，可为空或一句话描述",
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

	return []*schema.ToolInfo{
		{
			Name:        "search_products",
			Desc:        "搜索符合条件的商品，支持关键词、价格区间等筛选，可用于服饰搭配中各个单品的查找",
			ParamsOneOf: schema.NewParamsOneOfByParams(searchParams),
		},
	}
}
