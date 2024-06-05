/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"os"

	"github.com/spf13/cobra"
)

var version = "0.0.0-dev"

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Short:            "uplosi is a tool for uploading images to a cloud provider",
		PersistentPreRun: preRunRoot,
		Version:          version,
	}
	cmd.SetOut(os.Stdout)
	cmd.InitDefaultVersionFlag()
	cmd.AddCommand(newUploadCmd())
	cmd.AddCommand(newPrecalculateMeasurementsCmd())

	return cmd
}
