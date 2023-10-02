package template

import "strings"

func DefaultFuncMap() map[string]any {
	return map[string]any{
		"replaceAll": strings.ReplaceAll,
	}
}
