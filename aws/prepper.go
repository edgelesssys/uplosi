/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/
package aws

import (
	"context"
)

type Prepper struct{}

func (p *Prepper) Prepare(_ context.Context, imagePath, _ string) (string, error) {
	// AWS does not need any preparation.
	return imagePath, nil
}
