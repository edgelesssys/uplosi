/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"reflect"
	"slices"
	"strings"

	uplositemplate "github.com/edgelesssys/uplosi/template"

	"dario.cat/mergo"
)

var defaultConfig = Config{
	ImageVersion: "0.0.0",
	AWS: AWSConfig{
		ReplicationRegions: []string{},
		AMIName:            "{{.Name}}-{{.Version}}",
		AMIDescription:     "{{.Name}}-{{.Version}}",
		BlobName:           "{{.Name}}-{{.Version}}.raw",
		SnapshotName:       "{{.Name}}-{{.Version}}",
		Publish:            Some(false),
	},
	Azure: AzureConfig{
		AttestationVariant:  "azure-sev-snp",
		SharingProfile:      "community",
		ImageDefinitionName: "{{.Name}}",
		DiskName:            "{{.Name}}-{{.Version}}",
		Offer:               "Linux",
		SKU:                 "{{.Name}}-{{.VersionMajor}}",
		Publisher:           "Contoso",
	},
	GCP: GCPConfig{
		ImageName:   "{{.Name}}-{{replaceAll .Version \".\" \"-\"}}",
		ImageFamily: "{{.Name}}",
		BlobName:    "{{.Name}}-{{replaceAll .Version \".\" \"-\"}}.tar.gz",
	},
	OpenStack: OpenStackConfig{
		ImageName:  "{{.Name}}-{{.Version}}",
		Visibility: "public",
		Protected:  Some(false),
	},
}

type Config struct {
	Provider         string          `toml:"provider"`
	ImageVersion     string          `toml:"imageVersion"`
	ImageVersionFile string          `toml:"imageVersionFile"`
	Name             string          `toml:"name"`
	AWS              AWSConfig       `toml:"aws,omitempty"`
	Azure            AzureConfig     `toml:"azure,omitempty"`
	GCP              GCPConfig       `toml:"gcp,omitempty"`
	OpenStack        OpenStackConfig `toml:"openstack,omitempty"`
}

func (c *Config) Merge(other Config) error {
	return mergo.Merge(c, other, mergo.WithOverride, mergo.WithTransformers(&OptionTransformer{}))
}

func (c *Config) SetDefaults() error {
	return mergo.Merge(c, defaultConfig, mergo.WithTransformers(&OptionTransformer{}))
}

// Render renders the config by evaluating the version file and all template strings.
func (c *Config) Render(fileLookup func(name string) ([]byte, error)) error {
	if err := c.renderVersion(fileLookup); err != nil {
		return err
	}

	if err := c.renderTemplates(c); err != nil {
		return err
	}
	if err := c.renderTemplates(&c.AWS); err != nil {
		return err
	}
	if err := c.renderTemplates(&c.Azure); err != nil {
		return err
	}
	if err := c.renderTemplates(&c.GCP); err != nil {
		return err
	}
	if err := c.renderTemplates(&c.OpenStack); err != nil {
		return err
	}

	v := Validator{}

	if err := v.Validate(context.TODO(), *c); err != nil {
		return err
	}

	return nil
}

func (c *Config) renderVersion(fileLookup func(name string) ([]byte, error)) error {
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

func (c *Config) renderTemplates(configStruct any) error {
	numFields := reflect.TypeOf(configStruct).Elem().NumField()
	for i := 0; i < numFields; i++ {
		typeField := reflect.TypeOf(configStruct).Elem().Field(i)
		name := typeField.Name
		tag := typeField.Tag
		field := reflect.ValueOf(configStruct).Elem().Field(i)
		if err := c.renderFieldTemplate(name, field, tag); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) fieldTemplateData() fieldTemplateData {
	var VersionMajor, VersionMinor, VersionPatch string
	versionParts := strings.Split(c.ImageVersion, ".")
	if len(versionParts) == 3 {
		VersionMajor = versionParts[0]
		VersionMinor = versionParts[1]
		VersionPatch = versionParts[2]
	}
	return fieldTemplateData{
		Name:         c.Name,
		Version:      c.ImageVersion,
		VersionMajor: VersionMajor,
		VersionMinor: VersionMinor,
		VersionPatch: VersionPatch,
	}
}

func (c *Config) renderFieldTemplate(name string, field reflect.Value, tag reflect.StructTag) error {
	if tag.Get("template") != "true" {
		return nil
	}
	if field.Kind() != reflect.String || !field.CanSet() {
		return fmt.Errorf("field %s must be settable a string", name)
	}
	tmpl, err := template.New(name).Funcs(uplositemplate.DefaultFuncMap()).Parse(field.String())
	if err != nil {
		return err
	}
	renderedField := new(strings.Builder)
	if err := tmpl.Execute(renderedField, c.fieldTemplateData()); err != nil {
		return err
	}
	field.SetString(renderedField.String())
	return nil
}

type fieldTemplateData struct {
	Name         string
	Version      string
	VersionMajor string
	VersionMinor string
	VersionPatch string
}

type AWSConfig struct {
	Region                   string       `toml:"region,omitempty"`
	ReplicationRegions       []string     `toml:"replicationRegions,omitempty"`
	AMIName                  string       `toml:"amiName,omitempty" template:"true"`
	AMIDescription           string       `toml:"amiDescription,omitempty" template:"true"`
	Bucket                   string       `toml:"bucket,omitempty" template:"true"`
	BucketLocationConstraint string       `toml:"bucketLocationConstraint,omitempty" template:"false"`
	BlobName                 string       `toml:"blobName,omitempty" template:"true"`
	SnapshotName             string       `toml:"snapshotName,omitempty" template:"true"`
	Publish                  Option[bool] `toml:"publish,omitempty"`
}

type AzureConfig struct {
	SubscriptionID       string   `toml:"subscriptionID,omitempty"`
	Location             string   `toml:"location,omitempty"`
	ReplicationRegions   []string `toml:"replicationRegions,omitempty"`
	ResourceGroup        string   `toml:"resourceGroup,omitempty" template:"true"`
	AttestationVariant   string   `toml:"attestationVariant,omitempty" template:"true"`
	SharedImageGallery   string   `toml:"sharedImageGallery,omitempty" template:"true"`
	SharingProfile       string   `toml:"sharingProfile,omitempty" template:"true"`
	SharingNamePrefix    string   `toml:"sharingNamePrefix,omitempty" template:"true"`
	ImageDefinitionName  string   `toml:"imageDefinitionName,omitempty" template:"true"`
	Offer                string   `toml:"offer,omitempty" template:"true"`
	SKU                  string   `toml:"sku,omitempty" template:"true"`
	Publisher            string   `toml:"publisher,omitempty" template:"true"`
	DiskName             string   `toml:"diskName,omitempty" template:"true"`
	AdditionalSignatures []string `toml:"additionalSignatures,omitempty"`
}

type GCPConfig struct {
	Project     string `toml:"project,omitempty"`
	Location    string `toml:"location,omitempty"`
	ImageName   string `toml:"imageName,omitempty" template:"true"`
	ImageFamily string `toml:"imageFamily,omitempty" template:"true"`
	Bucket      string `toml:"bucket,omitempty" template:"true"`
	BlobName    string `toml:"blobName,omitempty" template:"true"`
}

type OpenStackConfig struct {
	Cloud      string            `toml:"cloud"`
	ImageName  string            `toml:"imageName,omitempty" template:"true"`
	Visibility string            `toml:"visibility,omitempty"`
	Hidden     Option[bool]      `toml:"hidden,omitempty"`
	Tags       []string          `toml:"tags,omitempty"`
	MinDiskGB  int               `toml:"minDiskGB,omitempty"`
	MinRamMB   int               `toml:"minRamMB,omitempty"`
	Protected  Option[bool]      `toml:"protected,omitempty"`
	Properties map[string]string `toml:"properties"`
}

type ConfigFile struct {
	Base     Config            `toml:"base"`
	Variants map[string]Config `toml:"variant"`
}

func (c *ConfigFile) Merge(other ConfigFile) error {
	if err := c.Base.Merge(other.Base); err != nil {
		return err
	}
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
	var vari Config
	if len(c.Variants) > 0 || len(name) > 0 {
		var ok bool
		vari, ok = c.Variants[name]
		if !ok {
			return Config{}, errors.New("variant not found")
		}
	}
	if err := out.Merge(c.Base); err != nil {
		return Config{}, err
	}
	if err := out.Merge(vari); err != nil {
		return Config{}, err
	}
	if err := out.SetDefaults(); err != nil {
		return Config{}, err
	}
	if err := out.Render(fileLookup); err != nil {
		return Config{}, err
	}

	return out, nil
}

func (c *ConfigFile) validateAll(fileLookup fileLookupFn, filters ...variantFilter) error {
	var errs error
	if len(c.Variants) == 0 {
		_, err := c.RenderedVariant(fileLookup, "")
		if err != nil {
			return fmt.Errorf("validating config: %w", err)
		}
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
		_, err := c.RenderedVariant(fileLookup, name)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("config for variant %s: %w", name, err))
		}
	}
	return errs
}

func (c *ConfigFile) ForEach(fn func(name string, cfg Config) error, fileLookup fileLookupFn, filters ...variantFilter) error {
	if err := c.validateAll(fileLookup, filters...); err != nil {
		return err
	}

	if len(c.Variants) == 0 {
		cfg, err := c.RenderedVariant(fileLookup, "")
		if err != nil {
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
