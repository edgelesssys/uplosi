/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/
package uploader

import (
	"errors"
	"io"
	"slices"
	"strings"

	"dario.cat/mergo"
	"golang.org/x/mod/semver"
)

type Config struct {
	Provider         string      `toml:"provider"`
	ImageVersion     string      `toml:"imageVersion"`
	ImageVersionFile string      `toml:"imageVersionFile"`
	Name             string      `toml:"name"`
	AWS              AWSConfig   `toml:"aws,omitempty"`
	Azure            AzureConfig `toml:"azure,omitempty"`
	GCP              GCPConfig   `toml:"gcp,omitempty"`
}

func (c *Config) Merge(other Config) error {
	return mergo.Merge(c, other, mergo.WithOverride, mergo.WithTransformers(&OptionTransformer{}))
}

func (c *Config) Validate() error {
	if len(c.Provider) == 0 {
		return errors.New("provider must be set")
	}
	if !semver.IsValid("v" + c.ImageVersion) {
		return errors.New("imageVersion must be of the form MAJOR.MINOR.PATCH")
	}
	if len(c.Name) == 0 {
		return errors.New("name must be set")
	}
	return nil
}

func (c *Config) RenderVersion(fileLookup func(name string) ([]byte, error)) error {
	if len(c.ImageVersionFile) == 0 {
		return nil
	}
	ver, err := fileLookup(c.ImageVersionFile)
	if err != nil {
		return err
	}
	c.ImageVersion = strings.TrimSpace(string(ver))
	return nil
}

type AWSConfig struct {
	Region             string       `toml:"region,omitempty"`
	ReplicationRegions []string     `toml:"replicationRegions,omitempty"`
	AMINameTemplate    string       `toml:"amiNameTemplate,omitempty"`
	AMIDescription     string       `toml:"amiDescription,omitempty"`
	Bucket             string       `toml:"bucket,omitempty"`
	BlobName           string       `toml:"blobName,omitempty"`
	SnapshotName       string       `toml:"snapshotName,omitempty"`
	Publish            Option[bool] `toml:"publish,omitempty"`
}

type AzureConfig struct {
	SubscriptionID              string `toml:"subscriptionID,omitempty"`
	Location                    string `toml:"location,omitempty"`
	ResourceGroup               string `toml:"resourceGroup,omitempty"`
	AttestationVariant          string `toml:"attestationVariant,omitempty"`
	SharedImageGalleryName      string `toml:"sharedImageGallery,omitempty"`
	SharingProfile              string `toml:"sharingProfile,omitempty"`
	SharingNamePrefix           string `toml:"sharingNamePrefix,omitempty"`
	ImageDefinitionNameTemplate string `toml:"imageDefinitionNameTemplate,omitempty"`
	Offer                       string `toml:"offer,omitempty"`
	SKU                         string `toml:"sku,omitempty"`
	Publisher                   string `toml:"publisher,omitempty"`
	DiskName                    string `toml:"diskName,omitempty"`
}

type GCPConfig struct {
	Project           string `toml:"project,omitempty"`
	Location          string `toml:"location,omitempty"`
	ImageNameTemplate string `toml:"imageNameTemplate,omitempty"`
	ImageFamily       string `toml:"imageFamily,omitempty"`
	Bucket            string `toml:"bucket,omitempty"`
	BlobName          string `toml:"blobName,omitempty"`
}

type Request struct {
	Image io.ReadSeekCloser
	Size  int64
}

type ConfigFile struct {
	Base     Config            `toml:"base"`
	Variants map[string]Config `toml:"variant"`
}

func (c *ConfigFile) Merge(other ConfigFile) error {
	c.Base.Merge(other.Base)
	if c.Variants == nil && len(other.Variants) > 0 {
		c.Variants = make(map[string]Config)
	}
	for k, v := range other.Variants {
		if _, ok := c.Variants[k]; !ok {
			c.Variants[k] = v
			continue
		}
		dst := c.Variants[k]
		if err := dst.Merge(v); err != nil {
			return err
		}
	}
	return nil
}

func (c *ConfigFile) RenderedVariant(fileLookup fileLookupFn, name string) (Config, error) {
	var out Config
	vari, ok := c.Variants[name]
	if !ok {
		return Config{}, errors.New("variant not found")
	}
	if err := out.Merge(c.Base); err != nil {
		return Config{}, err
	}
	if err := out.Merge(vari); err != nil {
		return Config{}, err
	}
	if err := out.RenderVersion(fileLookup); err != nil {
		return Config{}, err
	}

	return out, nil
}

func (c *ConfigFile) ForEach(fn func(name string, cfg Config) error, fileLookup fileLookupFn, filters ...variantFilter) error {
	if len(c.Variants) == 0 {
		cfg := Config{}
		if err := cfg.Merge(c.Base); err != nil {
			return err
		}
		if err := cfg.RenderVersion(fileLookup); err != nil {
			return err
		}
		return fn("", cfg)
	}
	variantNames := make([]string, 0, len(c.Variants))
	for name := range c.Variants {
		var filtered bool
		for _, filter := range filters {
			if !filter(name) {
				filtered = true
				break
			}
		}
		if filtered {
			continue
		}
		variantNames = append(variantNames, name)
	}
	slices.Sort(variantNames)
	for _, name := range variantNames {
		cfg, err := c.RenderedVariant(fileLookup, name)
		if err != nil {
			return err
		}
		if err := fn(name, cfg); err != nil {
			return err
		}
	}
	return nil
}

type fileLookupFn func(name string) ([]byte, error)

type variantFilter func(name string) bool
