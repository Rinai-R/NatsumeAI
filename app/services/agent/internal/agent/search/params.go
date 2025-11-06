package search

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Params struct {
	Keywords string
	MinPrice int64
	MaxPrice int64
	Limit    int32
	Slot     string
}

func ParseParams(raw map[string]any) Params {
	params := Params{}

	if v, ok := raw["keywords"]; ok {
		params.Keywords = joinStringSlice(v)
	}
	if v, ok := raw["min_price"]; ok {
		if parsed, ok := toInt64(v); ok {
			params.MinPrice = parsed
		}
	}
	if v, ok := raw["max_price"]; ok {
		if parsed, ok := toInt64(v); ok {
			params.MaxPrice = parsed
		}
	}
	if v, ok := raw["limit"]; ok {
		if parsed, ok := toInt32(v); ok {
			params.Limit = parsed
		}
	}
	if v, ok := raw["slot"]; ok {
		params.Slot = fmt.Sprint(v)
	}

	return params
}

func joinStringSlice(v any) string {
	slice := toStringSlice(v)
	return strings.Join(slice, ", ")
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
