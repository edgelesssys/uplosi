/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/

package azure

import (
	"context"
)

type Prepper struct{}

func (p *Prepper) Prepare(_ context.Context, imagePath, _ string) (string, error) {
	// Azure does not need any preparation.
	return imagePath, nil
}
