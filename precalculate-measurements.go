/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	measuredboot "github.com/edgelesssys/uplosi/measured-boot"
	"github.com/edgelesssys/uplosi/measured-boot/measure"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func newPrecalculateMeasurementsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "precalculate-measurements <image>",
		Short: "Precalculate TPM PCR measurements for an image",
		Args:  cobra.ExactArgs(1),
		RunE:  runPrecalculateMeasurements,
	}
	cmd.Flags().StringP("output-file", "o", "", "Output file for the precalculated measurements")
	cmd.Flags().StringP("uki-path", "u", measuredboot.UkiPath, "Path to the UKI file in the image")

	return cmd
}

func runPrecalculateMeasurements(cmd *cobra.Command, args []string) error {
	flags, err := parsePrecalculateMeasurementsFlags(cmd)
	if err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	fs := afero.NewOsFs()
	dissectToolchain := loadToolchain("DISSECT_TOOLCHAIN", "systemd-dissect")

	simulator, err := measuredboot.PrecalculatePCRs(fs, dissectToolchain, flags.ukiPath, args[0])
	if err != nil {
		return fmt.Errorf("precalculating PCRs: %w", err)
	}

	if flags.outputFile != "" {
		if err := writeOutput(fs, flags.outputFile, simulator); err != nil {
			return fmt.Errorf("writing output: %w", err)
		}
		cmd.Printf("Wrote precalculated measurements to %s\n", flags.outputFile)
	}

	return nil
}

type precalculateMeasurementsFlags struct {
	outputFile string
	ukiPath    string
}

func parsePrecalculateMeasurementsFlags(cmd *cobra.Command) (*precalculateMeasurementsFlags, error) {
	outputFile, err := cmd.Flags().GetString("output-file")
	if err != nil {
		return nil, fmt.Errorf("getting output-file flag: %w", err)
	}
	ukiPath, err := cmd.Flags().GetString("uki-path")
	if err != nil {
		return nil, fmt.Errorf("getting uki-path flag: %w", err)
	}
	return &precalculateMeasurementsFlags{
		outputFile: outputFile,
		ukiPath:    ukiPath,
	}, nil
}

func loadToolchain(key, fallback string) string {
	toolchain := os.Getenv(key)
	if toolchain == "" {
		toolchain = fallback
	}
	toolchain, err := exec.LookPath(toolchain)
	if err != nil {
		return ""
	}

	absolutePath, err := filepath.Abs(toolchain)
	if err != nil {
		return ""
	}
	return absolutePath
}

func writeOutput(fs afero.Fs, outputFile string, simulator *measure.Simulator) error {
	out, err := fs.Create(outputFile)
	if err != nil {
		return err
	}
	defer out.Close()

	return json.NewEncoder(out).Encode(simulator)
}
