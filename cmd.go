/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/edgelesssys/uplosi/aws"
	"github.com/edgelesssys/uplosi/azure"
	"github.com/edgelesssys/uplosi/config"
	"github.com/edgelesssys/uplosi/gcp"
	"github.com/edgelesssys/uplosi/openstack"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

const (
	configName = "uplosi.conf"
	configDir  = "uplosi.conf.d"
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

	return cmd
}

func newUploadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload <image>",
		Short: "Upload an image to a cloud provider",
		Args:  cobra.ExactArgs(1),
		RunE:  runUpload,
	}
	cmd.Flags().BoolP("increment-version", "i", false, "increment version number after upload")
	cmd.Flags().StringSlice("enable-variant-glob", []string{"*"}, "list of variant name globs to enable")
	cmd.Flags().StringSlice("disable-variant-glob", nil, "list of variant name globs to disable")
	cmd.Flags().StringP("config", "c", "", fmt.Sprintf("path to directory %s or %s resides in", configName, configDir))

	return cmd
}

func runUpload(cmd *cobra.Command, args []string) error {
	logger := log.New(cmd.ErrOrStderr(), "", log.LstdFlags)
	imagePath := args[0]

	flags, err := parseUploadFlags(cmd)
	if err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	conf, err := parseConfigFiles(flags.configPath)
	if err != nil {
		return fmt.Errorf("parsing config files: %w", err)
	}

	versionFiles := map[string][]byte{}
	versionFileLookup := func(name string) ([]byte, error) {
		if _, ok := versionFiles[name]; !ok {
			ver, err := os.ReadFile(name)
			if err != nil {
				return nil, fmt.Errorf("reading version file: %w", err)
			}
			versionFiles[name] = ver
		}
		return versionFiles[name], nil
	}

	allRefs := []string{}
	err = conf.ForEach(
		func(name string, cfg config.Config) error {
			refs, err := uploadVariant(cmd.Context(), imagePath, name, cfg, logger)
			if err != nil {
				return err
			}
			allRefs = append(allRefs, refs...)
			return nil
		},
		versionFileLookup,
		func(name string) bool {
			return filterGlobAny(flags.enableVariantGlobs, name)
		},
		func(name string) bool {
			return !filterGlobAny(flags.disableVariantGlobs, name)
		},
	)
	if err != nil {
		return fmt.Errorf("uploading variants: %w", err)
	}

	for _, ref := range allRefs {
		fmt.Println(ref)
	}

	if !flags.incrementVersion {
		return nil
	}
	if len(versionFiles) == 0 {
		return errors.New("increment-version flag set but no version files found")
	}
	for versionFileName, version := range versionFiles {
		newVer, err := incrementSemver(strings.TrimSpace(string(version)))
		if err != nil {
			return fmt.Errorf("incrementing semver: %w", err)
		}
		if err := writeVersionFile(versionFileName, []byte(newVer)); err != nil {
			return fmt.Errorf("writing version file: %w", err)
		}
	}
	return nil
}

func uploadVariant(ctx context.Context, imagePath, variant string, config config.Config, logger *log.Logger) ([]string, error) {
	var prepper Prepper
	var upload Uploader
	var err error

	if len(variant) > 0 {
		log.Println("Uploading variant", variant)
	}

	switch strings.ToLower(config.Provider) {
	case "aws":
		prepper = &aws.Prepper{}
		upload, err = aws.NewUploader(config, logger)
		if err != nil {
			return nil, fmt.Errorf("creating aws uploader: %w", err)
		}
	case "azure":
		prepper = &azure.Prepper{}
		upload, err = azure.NewUploader(config, logger)
		if err != nil {
			return nil, fmt.Errorf("creating azure uploader: %w", err)
		}
	case "gcp":
		prepper = &gcp.Prepper{}
		upload, err = gcp.NewUploader(config, logger)
		if err != nil {
			return nil, fmt.Errorf("creating gcp uploader: %w", err)
		}
	case "openstack":
		prepper = &openstack.Prepper{}
		upload, err = openstack.NewUploader(config, logger)
		if err != nil {
			return nil, fmt.Errorf("creating openstack uploader: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown provider: %s", config.Provider)
	}

	tmpDir, err := os.MkdirTemp("", "uplosi-")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	imagePath, err = prepper.Prepare(ctx, imagePath, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("preparing image: %w", err)
	}
	image, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("opening image: %w", err)
	}
	defer image.Close()
	imageFi, err := image.Stat()
	if err != nil {
		return nil, fmt.Errorf("getting image stats: %w", err)
	}

	refs, err := upload.Upload(ctx, image, imageFi.Size())
	if err != nil {
		return nil, fmt.Errorf("uploading image: %w", err)
	}

	return refs, nil
}

type uploadFlags struct {
	incrementVersion    bool
	enableVariantGlobs  []string
	disableVariantGlobs []string
	configPath          string
}

func parseUploadFlags(cmd *cobra.Command) (*uploadFlags, error) {
	incrementVersion, err := cmd.Flags().GetBool("increment-version")
	if err != nil {
		return nil, fmt.Errorf("getting increment-version flag: %w", err)
	}
	enableVariantGlobs, err := cmd.Flags().GetStringSlice("enable-variant-glob")
	if err != nil {
		return nil, fmt.Errorf("getting enable-variant-glob flag: %w", err)
	}
	disableVariantGlobs, err := cmd.Flags().GetStringSlice("disable-variant-glob")
	if err != nil {
		return nil, fmt.Errorf("getting disable-variant-glob flag: %w", err)
	}
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return nil, fmt.Errorf("getting config flag: %w", err)
	}
	return &uploadFlags{
		incrementVersion:    incrementVersion,
		enableVariantGlobs:  enableVariantGlobs,
		disableVariantGlobs: disableVariantGlobs,
		configPath:          configPath,
	}, nil
}

func filterGlobAny(globs []string, name string) bool {
	for _, glob := range globs {
		if ok, _ := filepath.Match(glob, name); ok {
			return true
		}
	}
	return false
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

func writeVersionFile(path string, data []byte) error {
	versionFile, err := os.OpenFile(path, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer versionFile.Close()
	if _, err := versionFile.Write(data); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	return nil
}

type Prepper interface {
	Prepare(ctx context.Context, imagePath, tmpDir string) (string, error)
}

type Uploader interface {
	Upload(ctx context.Context, image io.ReadSeeker, size int64) (refs []string, retErr error)
}

func parseConfigFiles(configPath string) (*config.ConfigFile, error) {
	configLocation := configName
	configDirLocation := configDir
	if configPath != "" {
		configLocation = path.Join(configPath, configName)
		configDirLocation = path.Join(configPath, configDir)
	}

	var conf config.ConfigFile
	if err := readTOMLFile(configLocation, &conf); err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	dirEntries, err := os.ReadDir(configDirLocation)
	if os.IsNotExist(err) {
		return &conf, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config dir: %w", err)
	}
	for _, dirEntry := range dirEntries {
		var cfgOverlay config.ConfigFile
		if dirEntry.IsDir() {
			continue
		}
		if filepath.Ext(dirEntry.Name()) != ".conf" {
			continue
		}
		if err := readTOMLFile(filepath.Join(configDir, dirEntry.Name()), &cfgOverlay); err != nil {
			return nil, fmt.Errorf("reading config: %w", err)
		}
		if err := conf.Merge(cfgOverlay); err != nil {
			return nil, fmt.Errorf("merging config: %w", err)
		}
	}
	return &conf, nil
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
