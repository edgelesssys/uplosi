/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/edgelesssys/uplosi/aws"
	"github.com/edgelesssys/uplosi/azure"
	"github.com/edgelesssys/uplosi/gcp"
	"github.com/edgelesssys/uplosi/uploader"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

const (
	configName = "uplosi.toml"
)

func newCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "uplosi <provider> <image>",
		Short:            "uplosi is a tool for uploading images to a cloud provider",
		PersistentPreRun: preRunRoot,
		RunE:             run,
		Args:             cobra.MatchAll(cobra.ExactArgs(2), isCSP(0)),
	}
	cmd.SetOut(os.Stdout)

	cmd.Flags().BoolP("increment-version", "i", false, "increment version number in config after upload")

	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	logger := log.New(cmd.OutOrStderr(), "", log.LstdFlags)
	provider := args[0]
	imagePath := args[1]

	flags, err := parseUploadFlags(cmd)
	if err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	var config uploader.Config
	if err := readTOMLFile(configName, &config); err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	var prepper Prepper
	var upload Uploader

	switch strings.ToLower(provider) {
	case "aws":
		prepper = &aws.Prepper{}
		upload, err = aws.NewUploader(config, logger)
		if err != nil {
			return fmt.Errorf("creating aws uploader: %w", err)
		}
	case "azure":
		prepper = &azure.Prepper{}
		upload, err = azure.NewUploader(config, logger)
		if err != nil {
			return fmt.Errorf("creating azure uploader: %w", err)
		}
	case "gcp":
		prepper = &gcp.Prepper{}
		upload, err = gcp.NewUploader(config, logger)
		if err != nil {
			return fmt.Errorf("creating gcp uploader: %w", err)
		}
	}

	tmpDir, err := os.MkdirTemp("", "uplosi-")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	imagePath, err = prepper.Prepare(cmd.Context(), imagePath, tmpDir)
	if err != nil {
		return fmt.Errorf("preparing image: %w", err)
	}
	image, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("opening image: %w", err)
	}
	defer image.Close()
	imageFi, err := image.Stat()
	if err != nil {
		return fmt.Errorf("getting image stats: %w", err)
	}

	req := &uploader.Request{
		Image: image,
		Size:  imageFi.Size(),
	}
	refs, err := upload.Upload(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("uploading image: %w", err)
	}

	for _, ref := range refs {
		fmt.Println(ref)
	}

	if flags.incrementVersion {
		newVer, err := incrementSemver(config.ImageVersion)
		if err != nil {
			return fmt.Errorf("incrementing semver: %w", err)
		}
		config.ImageVersion = newVer
		if err := writeTOMLFile(configName, config); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}
	}

	return nil
}

type uploadFlags struct {
	incrementVersion bool
}

func parseUploadFlags(cmd *cobra.Command) (*uploadFlags, error) {
	incrementVersion, err := cmd.Flags().GetBool("increment-version")
	if err != nil {
		return nil, fmt.Errorf("getting increment-version flag: %w", err)
	}
	return &uploadFlags{
		incrementVersion: incrementVersion,
	}, nil
}

func readTOMLFile(path string, data any) error {
	configFile, err := os.OpenFile(path, os.O_RDONLY, os.ModeAppend)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer configFile.Close()
	if _, err := toml.NewDecoder(configFile).Decode(data); err != nil {
		return fmt.Errorf("decoding file: %w", err)
	}
	return nil
}

func writeTOMLFile(path string, data any) error {
	configFile, err := os.OpenFile(path, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer configFile.Close()
	if err := toml.NewEncoder(configFile).Encode(data); err != nil {
		return fmt.Errorf("encoding file: %w", err)
	}
	return nil
}

func supportedCSPs() []string {
	return []string{"aws", "azure", "gcp"}
}

func isCSP(position int) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		for _, csp := range supportedCSPs() {
			if args[position] == csp {
				return nil
			}
		}
		return fmt.Errorf("unsupported cloud service provider: %s", args[position])
	}
}

type Prepper interface {
	Prepare(ctx context.Context, imagePath, tmpDir string) (string, error)
}

type Uploader interface {
	Upload(ctx context.Context, req *uploader.Request) (refs []string, retErr error)
}

func canonicalSemver(version string) error {
	ver := "v" + version
	if !semver.IsValid(ver) {
		return fmt.Errorf("invalid semver: %s", version)
	}
	if semver.Canonical(ver) != ver {
		return fmt.Errorf("not canonical semver: %s", version)
	}
	return nil
}

func incrementSemver(version string) (string, error) {
	canonical := strings.TrimPrefix(semver.Canonical("v"+version), "v")
	parts := strings.Split(canonical, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("splitting canonical version: %s, %v", canonical, parts)
	}

	patch := parts[2]
	patchNum, err := strconv.Atoi(patch)
	if err != nil {
		return "", fmt.Errorf("converting patch number: %w", err)
	}

	patchNum++
	return fmt.Sprintf("%s.%s.%d", parts[0], parts[1], patchNum), nil
}
