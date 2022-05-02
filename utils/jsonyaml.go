package utils

import "fmt"

func YamlToJson(m map[any]any) map[string]any {
	res := map[string]any{}

	for k, v := range m {
		var ks string
		switch k2 := k.(type) {
		case string:
			ks = k2
		case *string:
			ks = *k2
		default:
			ks = fmt.Sprint(k2)
		}
		res[ks] = yamlToJsonValue(v)
	}
	return res
}

func yamlToJsonValue(v any) any {
	switch v2 := v.(type) {
	case []any:
		return yamlToJsonArr(v2)
	case map[any]any:
		return YamlToJson(v2)
	default:
		return v
	}
}

func yamlToJsonArr(v []any) []any {
	arr := make([]any, len(v))
	for i := range v {
		arr[i] = yamlToJsonValue(v[i])
	}
	return arr
}
