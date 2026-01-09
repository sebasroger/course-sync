package devutil

import "encoding/json"

// pick toma cualquier struct/map, lo pasa a map[string]any vía JSON,
// y devuelve solo las keys pedidas. Útil para debug/prints.
func pick(v any, keys ...string) map[string]any {
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]any{}
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]any{}
	}

	out := make(map[string]any, len(keys))
	for _, k := range keys {
		if val, ok := m[k]; ok {
			out[k] = val
		}
	}
	return out
}

func Pick(v any, keys ...string) map[string]any {
	return pick(v, keys...)
}
