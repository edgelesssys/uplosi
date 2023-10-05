/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/
package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigRenderVersionFromFile(t *testing.T) {
	assert := assert.New(t)
	lookup := stubFileLookup{
		"image-version.txt": []byte("0.0.2"),
	}
	config := Config{
		Name:             "test",
		ImageVersion:     "0.0.1", // this will be overwritten by the file
		ImageVersionFile: "image-version.txt",
	}
	assert.NoError(config.Render(lookup.Lookup))
	assert.Equal("0.0.2", config.ImageVersion)
}

func TestConfigRenderTemplate(t *testing.T) {
	assert := assert.New(t)
	lookup := stubFileLookup{}
	config := Config{
		Name:         "name",
		ImageVersion: "0.0.1",
		GCP: GCPConfig{
			ImageName: "prefix-{{.Name}}-{{replaceAll .Version \".\" \"-\"}}-suffix",
		},
	}
	assert.NoError(config.Render(lookup.Lookup))
	assert.Equal("prefix-name-0-0-1-suffix", config.GCP.ImageName)
}

func TestConfigSetDefaults(t *testing.T) {
	assert := assert.New(t)
	config := Config{
		ImageVersion: "0.0.1", // this has a value and will not be overwritten with the default
		Name:         "",      // this has no default and will stay empty
		AWS: AWSConfig{
			Publish: None[bool](), // this will be set to the default
		},
		Azure: AzureConfig{
			SharingProfile: "", // this will be set to the default
		},
	}
	assert.NoError(config.SetDefaults())
	assert.Equal("0.0.1", config.ImageVersion)
	assert.Empty(config.Name)
	assert.True(config.AWS.Publish.IsSome())
	assert.False(config.AWS.Publish.Val)
	assert.Equal("community", config.Azure.SharingProfile)
}

func TestConfigMerge(t *testing.T) {
	assert := assert.New(t)
	dst := Config{}
	src := fullConfig()

	assert.NoError(dst.Merge(src))
	assert.Equal(src, dst)

	dst = Config{}
	dst.AWS.Publish = Some(false)
	assert.NoError(dst.Merge(src))
	assert.True(dst.AWS.Publish.IsSome())
	assert.False(dst.AWS.Publish.Unwrap())

	dst = Config{}
	dst.AWS.Publish = Some(false)
	src = fullConfig()
	src.AWS.Publish = None[bool]()
	assert.NoError(dst.Merge(src))
	assert.True(dst.AWS.Publish.IsSome())
	assert.False(dst.AWS.Publish.Unwrap())

	dst = Config{}
	dst.AWS.Publish = Some(true)
	src = fullConfig()
	src.AWS.Publish = None[bool]()
	assert.NoError(dst.Merge(src))
	assert.True(dst.AWS.Publish.IsSome())
	assert.True(dst.AWS.Publish.Unwrap())

	dst = Config{}
	dst.AWS.Publish = Some(true)
	src = fullConfig()
	src.AWS.Publish = Some(false)
	assert.NoError(dst.Merge(src))
	assert.True(dst.AWS.Publish.IsSome())
	assert.True(dst.AWS.Publish.Unwrap())
}

func TestConfigFileMerge(t *testing.T) {
	assert := assert.New(t)
	dst := ConfigFile{}
	src := fullConfigFile()

	assert.NoError(dst.Merge(src))
	assert.Equal(src, dst)

	dst = ConfigFile{
		Base: Config{
			Name: "base",
		},
	}
	src = fullConfigFile()
	src.Base.Name = ""
	assert.NoError(dst.Merge(src))
	assert.Equal("base", dst.Base.Name)

	dst = ConfigFile{
		Variants: map[string]Config{
			"a": {
				Name: "a",
			},
		},
	}
	src = fullConfigFile()
	srcVariant := src.Variants["a"]
	srcVariant.Name = ""
	assert.NoError(dst.Merge(src))
	assert.Equal("a", dst.Variants["a"].Name)
	assert.Equal("test", dst.Variants["b"].Name)
}

type stubFileLookup map[string][]byte

func (s stubFileLookup) Lookup(name string) ([]byte, error) {
	val, ok := s[name]
	if !ok {
		return nil, errors.New("not found")
	}
	return val, nil
}

func fullConfig() Config {
	return Config{
		ImageVersion: "0.0.1",
		Name:         "test",
		AWS: AWSConfig{
			Region:             "eu-central-1",
			ReplicationRegions: []string{"eu-west-1", "eu-west-2"},
			AMIName:            "ami-name-template",
			AMIDescription:     "ami-description",
			Bucket:             "bucket",
			BlobName:           "blob-name",
			SnapshotName:       "snapshot-name",
			Publish:            Some[bool](true),
		},
		Azure: AzureConfig{
			SubscriptionID:         "subscription-id",
			Location:               "location",
			ResourceGroup:          "resource-group",
			AttestationVariant:     "attestation-variant",
			SharedImageGalleryName: "shared-image-gallery",
			SharingProfile:         "sharing-profile",
			SharingNamePrefix:      "sharing-name-prefix",
			ImageDefinitionName:    "image-definition-name-template",
			Offer:                  "offer",
			SKU:                    "sku",
			Publisher:              "publisher",
			DiskName:               "disk-name",
		},
		GCP: GCPConfig{
			Project:     "project",
			Location:    "location",
			ImageName:   "image-name-template",
			ImageFamily: "image-family",
			Bucket:      "bucket",
			BlobName:    "blob-name",
		},
	}
}

func fullConfigFile() ConfigFile {
	return ConfigFile{
		Base: fullConfig(),
		Variants: map[string]Config{
			"a": fullConfig(),
			"b": fullConfig(),
		},
	}
}
