/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/

package openstack

import (
	"context"
)

type Prepper struct{}

func (p *Prepper) Prepare(_ context.Context, imagePath, _ string) (string, error) {
	// OpenStack does not need any preparation.
	return imagePath, nil
}
