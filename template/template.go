/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/

package template

import "strings"

func DefaultFuncMap() map[string]any {
	return map[string]any{
		"replaceAll": strings.ReplaceAll,
	}
}
