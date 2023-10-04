/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/
package uploader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func fullConfig() Config {
	return Config{
		ImageVersion: "0.0.1",
		Name:         "test",
		AWS: AWSConfig{
			Region:             "eu-central-1",
			ReplicationRegions: []string{"eu-west-1", "eu-west-2"},
			AMINameTemplate:    "ami-name-template",
			AMIDescription:     "ami-description",
			Bucket:             "bucket",
			BlobName:           "blob-name",
			SnapshotName:       "snapshot-name",
			Publish:            Some[bool](true),
		},
		Azure: AzureConfig{
			SubscriptionID:              "subscription-id",
			Location:                    "location",
			ResourceGroup:               "resource-group",
			AttestationVariant:          "attestation-variant",
			SharedImageGalleryName:      "shared-image-gallery",
			SharingProfile:              "sharing-profile",
			SharingNamePrefix:           "sharing-name-prefix",
			ImageDefinitionNameTemplate: "image-definition-name-template",
			Offer:                       "offer",
			SKU:                         "sku",
			Publisher:                   "publisher",
			DiskName:                    "disk-name",
		},
		GCP: GCPConfig{
			Project:           "project",
			Location:          "location",
			ImageNameTemplate: "image-name-template",
			ImageFamily:       "image-family",
			Bucket:            "bucket",
			BlobName:          "blob-name",
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
