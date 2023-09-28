/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/
package uploader

import (
	"io"
)

type Config struct {
	ImageVersion string      `yaml:"imageVersion"`
	Name         string      `yaml:"name"`
	Azure        AzureConfig `yaml:"azure"`
}

type AzureConfig struct {
	SubscriptionID         string `yaml:"subscriptionID"`
	Location               string `yaml:"location"`
	ResourceGroup          string `yaml:"resourceGroup"`
	AttestationVariant     string `yaml:"attestationVariant"`
	SharedImageGalleryName string `yaml:"sharedImageGallery"`
	SharingProfile         string `yaml:"sharingProfile"`
	SharingNamePrefix      string `yaml:"sharingNamePrefix"`
	ImageDefinitionName    string `yaml:"imageDefinition"`
	Offer                  string `yaml:"offer"`
	SKU                    string `yaml:"sku"`
	Publisher              string `yaml:"publisher"`
	DiskName               string `yaml:"diskName"`
}

type Request struct {
	Image     io.ReadSeekCloser
	Timestamp string
	Size      int64
}
