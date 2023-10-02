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
	AWS          AWSConfig   `toml:"aws"`
	Azure        AzureConfig `toml:"azure"`
	GCP          GCPConfig   `toml:"gcp"`
}

type AWSConfig struct {
	Region             string   `toml:"region"`
	ReplicationRegions []string `toml:"replicationRegions"`
	AMINameTemplate    string   `toml:"amiNameTemplate"`
	AMIDescription     string   `toml:"amiDescription"`
	Bucket             string   `toml:"bucket"`
	BlobName           string   `toml:"blobName"`
	SnapshotName       string   `toml:"snapshotName"`
	Publish            bool     `toml:"publish"`
}

type AzureConfig struct {
	SubscriptionID              string `toml:"subscriptionID"`
	Location                    string `toml:"location"`
	ResourceGroup               string `toml:"resourceGroup"`
	AttestationVariant          string `toml:"attestationVariant"`
	SharedImageGalleryName      string `toml:"sharedImageGallery"`
	SharingProfile              string `toml:"sharingProfile"`
	SharingNamePrefix           string `toml:"sharingNamePrefix"`
	ImageDefinitionNameTemplate string `toml:"imageDefinitionNameTemplate"`
	Offer                       string `toml:"offer"`
	SKU                         string `toml:"sku"`
	Publisher                   string `toml:"publisher"`
	DiskName                    string `toml:"diskName"`
}

type GCPConfig struct {
	Project           string `toml:"project"`
	Location          string `toml:"location"`
	ImageNameTemplate string `toml:"imageNameTemplate"`
	ImageFamily       string `toml:"imageFamily"`
	Bucket            string `toml:"bucket"`
	BlobName          string `toml:"blobName"`
}

type Request struct {
	Image io.ReadSeekCloser
	Size  int64
}
