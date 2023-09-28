/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/
package azure

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Prepper struct{}

func (p *Prepper) Prepare(ctx context.Context, imagePath string) (string, error) {
	// Azure needs image padded to next MiB.
	// TODO(katexochen): Ideally this should be padded correctly by mkosi/repart,
	// or operate on a copy.
	cmd := exec.CommandContext(ctx,
		"truncate",
		"--size", "%1MiB",
		imagePath,
	)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}

	targetPath := strings.TrimSuffix(imagePath, filepath.Ext(imagePath)) + ".vhd"
	cmd = exec.CommandContext(ctx,
		"qemu-img",
		"convert",
		"-f", "raw",
		"-O", "vpc",
		"-o", "force_size,subformat=fixed",
		imagePath,
		targetPath,
	)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return targetPath, nil
}
