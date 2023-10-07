/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	testCases := map[string]struct {
		base      Config
		overrides Config
		mutation  func(*Config)
		wantErr   bool
	}{
		"empty config": {
			wantErr: true,
		},
		"full config": {
			base: validConfig(),
		},
		"valid AWS config": {
			base:      validConfig(),
			overrides: Config{Provider: "aws"},
		},
		"valid Azure config": {
			base:      validConfig(),
			overrides: Config{Provider: "azure"},
		},
		"valid GCP config": {
			base:      validConfig(),
			overrides: Config{Provider: "gcp"},
		},
		"unknown provider": {
			base:      validConfig(),
			overrides: Config{Provider: "foo"},
			wantErr:   true,
		},
		"missing version": {
			base:     validConfig(),
			mutation: func(c *Config) { c.ImageVersion = "" },
			wantErr:  true,
		},
		"invalid version": {
			base:      validConfig(),
			overrides: Config{ImageVersion: "v1.2.3-dev"},
			wantErr:   true,
		},
		"missing name": {
			base:     validConfig(),
			mutation: func(c *Config) { c.Name = "" },
			wantErr:  true,
		},
		"missing AWS region": {
			base:      validConfig(),
			overrides: Config{Provider: "aws"},
			mutation: func(c *Config) {
				c.AWS.Region = ""
			},
			wantErr: true,
		},
		"missing AWS amiName": {
			base:      validConfig(),
			overrides: Config{Provider: "aws"},
			mutation: func(c *Config) {
				c.AWS.AMIName = ""
			},
			wantErr: true,
		},
		"wrong length AWS amiName": {
			base: validConfig(),
			overrides: Config{
				Provider: "aws",
				AWS: AWSConfig{
					AMIName: strings.Repeat("a", 129),
				},
			},
			wantErr: true,
		},
		"invalid AWS amiName": {
			base: validConfig(),
			overrides: Config{
				Provider: "aws",
				AWS: AWSConfig{
					AMIName: "invalid<ami_name",
				},
			},
			wantErr: true,
		},
		"missing AWS blobName": {
			base: validConfig(),
			overrides: Config{
				Provider: "aws",
			},
			mutation: func(c *Config) {
				c.AWS.BlobName = ""
			},
			wantErr: true,
		},
		"missing AWS snapshotName": {
			base: validConfig(),
			overrides: Config{
				Provider: "aws",
			},
			mutation: func(c *Config) {
				c.AWS.SnapshotName = ""
			},
			wantErr: true,
		},
		"uninitialized AWS Publish setting": {
			base: validConfig(),
			overrides: Config{
				Provider: "aws",
			},
			mutation: func(c *Config) {
				c.AWS.Publish = None[bool]()
			},
			wantErr: true,
		},
		"missing Azure subscriptionID": {
			base: validConfig(),
			overrides: Config{
				Provider: "azure",
			},
			mutation: func(c *Config) {
				c.Azure.SubscriptionID = ""
			},
			wantErr: true,
		},
		"invalid Azure subscriptionID": {
			base: validConfig(),
			overrides: Config{
				Provider: "azure",
				Azure:    AzureConfig{SubscriptionID: "invalid"},
			},
			wantErr: true,
		},
		"missing Azure location": {
			base: validConfig(),
			overrides: Config{
				Provider: "azure",
			},
			mutation: func(c *Config) {
				c.Azure.Location = ""
			},
			wantErr: true,
		},
		"missing Azure resourceGroup": {
			base: validConfig(),
			overrides: Config{
				Provider: "azure",
			},
			mutation: func(c *Config) {
				c.Azure.ResourceGroup = ""
			},
			wantErr: true,
		},
		"missing Azure attestationVariant": {
			base: validConfig(),
			overrides: Config{
				Provider: "azure",
			},
			mutation: func(c *Config) {
				c.Azure.AttestationVariant = ""
			},
			wantErr: true,
		},
		"invalid Azure attestationVariant": {
			base: validConfig(),
			overrides: Config{
				Provider: "azure",
				Azure:    AzureConfig{AttestationVariant: "invalid"},
			},
			wantErr: true,
		},
		"missing Azure sharedImageGallery": {
			base: validConfig(),
			overrides: Config{
				Provider: "azure",
			},
			mutation: func(c *Config) {
				c.Azure.SharedImageGallery = ""
			},
			wantErr: true,
		},
		"missing Azure sharingProfile": {
			base: validConfig(),
			overrides: Config{
				Provider: "azure",
			},
			mutation: func(c *Config) {
				c.Azure.SharedImageGallery = ""
			},
			wantErr: true,
		},
		"invalid Azure sharingProfile": {
			base: validConfig(),
			overrides: Config{
				Provider: "azure",
				Azure:    AzureConfig{SharingProfile: "invalid"},
			},
			wantErr: true,
		},
		"missing Azure sharingNamePrefix with sharingProfile community": {
			base: validConfig(),
			overrides: Config{
				Provider: "azure",
				Azure: AzureConfig{
					SharingProfile: "community",
				},
			},
			mutation: func(c *Config) {
				c.Azure.SharingNamePrefix = ""
			},
			wantErr: true,
		},
		"missing Azure sharingNamePrefix with sharingProfile private": {
			base: validConfig(),
			overrides: Config{
				Provider: "azure",
				Azure: AzureConfig{
					SharingProfile: "private",
				},
			},
			mutation: func(c *Config) {
				c.Azure.SharingNamePrefix = ""
			},
		},
		"missing Azure imageDefinitionName": {
			base: validConfig(),
			overrides: Config{
				Provider: "azure",
			},
			mutation: func(c *Config) {
				c.Azure.ImageDefinitionName = ""
			},
			wantErr: true,
		},
		"missing Azure diskName": {
			base: validConfig(),
			overrides: Config{
				Provider: "azure",
			},
			mutation: func(c *Config) {
				c.Azure.DiskName = ""
			},
			wantErr: true,
		},
		"missing GCP project": {
			base: validConfig(),
			overrides: Config{
				Provider: "gcp",
			},
			mutation: func(c *Config) {
				c.GCP.Project = ""
			},
			wantErr: true,
		},
		"invalid GCP project": {
			base: validConfig(),
			overrides: Config{
				Provider: "gcp",
				GCP: GCPConfig{
					Project: "-invalid project name",
				},
			},
			wantErr: true,
		},
		"missing GCP location": {
			base: validConfig(),
			overrides: Config{
				Provider: "gcp",
			},
			mutation: func(c *Config) {
				c.GCP.Location = ""
			},
			wantErr: true,
		},
		"missing GCP imageName": {
			base: validConfig(),
			overrides: Config{
				Provider: "gcp",
			},
			mutation: func(c *Config) {
				c.GCP.ImageName = ""
			},
			wantErr: true,
		},
		"missing GCP imageFamily": {
			base: validConfig(),
			overrides: Config{
				Provider: "gcp",
			},
			mutation: func(c *Config) {
				c.GCP.ImageFamily = ""
			},
			wantErr: true,
		},
		"missing GCP bucket": {
			base: validConfig(),
			overrides: Config{
				Provider: "gcp",
			},
			mutation: func(c *Config) {
				c.GCP.Bucket = ""
			},
			wantErr: true,
		},
		"missing GCP blobName": {
			base: validConfig(),
			overrides: Config{
				Provider: "gcp",
			},
			mutation: func(c *Config) {
				c.GCP.BlobName = ""
			},
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			ctx := context.Background()

			v := Validator{}

			cfg := Config{}
			assert.NoError(cfg.Merge(tc.base))
			assert.NoError(cfg.Merge(tc.overrides))
			if tc.mutation != nil {
				tc.mutation(&cfg)
			}

			err := v.Validate(ctx, cfg)
			if err != nil {
				t.Log(err)
			}
			if tc.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
		})
	}
}

func validConfig() Config {
	return Config{
		Provider:     "aws",
		ImageVersion: "0.0.0",
		Name:         "my-image",
		AWS: AWSConfig{
			Region:             "us-east-1",
			ReplicationRegions: []string{"us-west-1", "us-west-2"},
			AMIName:            "my-ami(123).my/ami_name",
			AMIDescription:     "my-ami-description",
			Bucket:             "my-bucket",
			BlobName:           "my-blob",
			SnapshotName:       "my-snapshot",
			Publish:            Some[bool](true),
		},
		Azure: AzureConfig{
			SubscriptionID:      "00000000-0000-0000-0000-000000000000",
			Location:            "westeurope",
			ResourceGroup:       "my-resource-group",
			AttestationVariant:  "azure-sev-snp",
			SharedImageGallery:  "mygallery",
			SharingProfile:      "community",
			SharingNamePrefix:   "prefix",
			ImageDefinitionName: "my-image",
			Offer:               "Linux",
			SKU:                 "my-sku",
			Publisher:           "Contoso",
			DiskName:            "my-disk",
		},
		GCP: GCPConfig{
			Project:     "my-project",
			Location:    "us-central1",
			ImageName:   "my-image",
			ImageFamily: "my-family",
			Bucket:      "my-bucket",
			BlobName:    "my-blob",
		},
	}
}
