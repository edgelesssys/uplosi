/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/
package uploader

import (
	"io"
)

type Config struct {
	ImageVersion string      `toml:"imageVersion"`
	Name         string      `toml:"name"`
	Azure        AzureConfig `toml:"azure"`
}

type AzureConfig struct {
	SubscriptionID         string `toml:"subscriptionID"`
	Location               string `toml:"location"`
	ResourceGroup          string `toml:"resourceGroup"`
	AttestationVariant     string `toml:"attestationVariant"`
	SharedImageGalleryName string `toml:"sharedImageGallery"`
	SharingProfile         string `toml:"sharingProfile"`
	SharingNamePrefix      string `toml:"sharingNamePrefix"`
	ImageDefinitionName    string `toml:"imageDefinition"`
	Offer                  string `toml:"offer"`
	SKU                    string `toml:"sku"`
	Publisher              string `toml:"publisher"`
	DiskName               string `toml:"diskName"`
}

type Request struct {
	Image     io.ReadSeekCloser
	Timestamp string
	Size      int64
}
