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
	"time"

	"github.com/edgelesssys/uplosi/azure"
	"github.com/edgelesssys/uplosi/uploader"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	configName      = "upload-image.yml"
	timestampFormat = "20060102150405"
)

func newCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "upload-image",
		Short:            "upload-image is a tool for uploading images to a cloud provider",
		PersistentPreRun: preRunRoot,
		RunE:             run,
		Args:             cobra.MatchAll(cobra.ExactArgs(2), isCSP(0)),
	}
	cmd.SetOut(os.Stdout)

	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	provider := args[0]
	imagePath := args[1]
	logger := log.New(cmd.OutOrStderr(), "", log.LstdFlags)

	configFile, err := os.Open(configName)
	if err != nil {
		return fmt.Errorf("opening config file: %w", err)
	}
	defer configFile.Close()
	var config uploader.Config
	if err := yaml.NewDecoder(configFile).Decode(&config); err != nil {
		return fmt.Errorf("decoding config file: %w", err)
	}

	var prepper Prepper
	var upload Uploader

	switch provider {
	case "azure":
		prepper = &azure.Prepper{}
		upload, err = azure.NewUploader(config, logger)
		if err != nil {
			return fmt.Errorf("creating azure uploader: %w", err)
		}
	}

	imagePath, err = prepper.Prepare(cmd.Context(), imagePath)
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
		Image:     image,
		Timestamp: time.Now().UTC().Format(timestampFormat),
		Size:      imageFi.Size(),
	}
	ref, err := upload.Upload(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("uploading image: %w", err)
	}

	fmt.Println(ref)
	return nil
}

func supportedCSPs() []string {
	return []string{"azure"}
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
	Prepare(ctx context.Context, imagePath string) (string, error)
}

type Uploader interface {
	Upload(ctx context.Context, req *uploader.Request) (ref string, retErr error)
}
