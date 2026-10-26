package core

import (
	"encoding/json"
)

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func getArgs(req interface{}) map[string]any {
	type getArgser interface {
		GetArguments() map[string]any
	}
	if g, ok := req.(getArgser); ok {
		if a := g.GetArguments(); a != nil {
			return a
		}
	}
	return map[string]any{}
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case float64:
			return int(t)
		case int:
			return t
		case int64:
			return int(t)
		}
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
