/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/

package openstack

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/edgelesssys/uplosi/config"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/imagedata"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/utils/openstack/clientconfig"
)

const microversion = "2.42"

type Uploader struct {
	config config.Config

	image func(context.Context) (*gophercloud.ServiceClient, error)

	log *log.Logger
}

func NewUploader(config config.Config, log *log.Logger) (*Uploader, error) {
	clientOpts := &clientconfig.ClientOpts{
		Cloud: config.OpenStack.Cloud,
	}

	return &Uploader{
		config: config,
		image: func(ctx context.Context) (*gophercloud.ServiceClient, error) {
			imageClient, err := clientconfig.NewServiceClient("image", clientOpts)
			if err != nil {
				return nil, err
			}
			imageClient.Microversion = microversion
			return imageClient, nil
		},
		log: log,
	}, nil
}

func (u *Uploader) Upload(ctx context.Context, image io.ReadSeeker, _ int64) (refs []string, retErr error) {
	if err := u.ensureImageDeleted(ctx); err != nil {
		return nil, fmt.Errorf("pre-cleaning: ensuring no image using the same name exists: %w", err)
	}
	imageID, err := u.createImage(ctx, image)
	if err != nil {
		return nil, fmt.Errorf("creating image: %w", err)
	}
	return []string{imageID}, nil
}

func (u *Uploader) createImage(ctx context.Context, image io.ReadSeeker) (string, error) {
	visibility := images.ImageVisibility(u.config.OpenStack.Visibility)
	if visibility == images.ImageVisibility("") {
		visibility = images.ImageVisibilityPublic
	}
	protected := u.config.OpenStack.Protected.UnwrapOr(false)
	hidden := u.config.OpenStack.Hidden.UnwrapOr(false)
	createOpts := images.CreateOpts{
		Name:            u.config.OpenStack.ImageName,
		ContainerFormat: "bare",
		DiskFormat:      "raw",
		Visibility:      &visibility,
		Hidden:          &hidden,
		Tags:            u.config.OpenStack.Tags,
		Protected:       &protected,
		MinDisk:         u.config.OpenStack.MinDiskGB,
		MinRAM:          u.config.OpenStack.MinRamMB,
		Properties:      u.config.OpenStack.Properties,
	}

	imageClient, err := u.image(ctx)
	if err != nil {
		return "", err
	}

	u.log.Printf("Creating image %q", u.config.OpenStack.ImageName)

	newImage, err := images.Create(imageClient, createOpts).Extract()
	if err != nil {
		return "", fmt.Errorf("creating image: %w", err)
	}

	if err := imagedata.Upload(imageClient, newImage.ID, image).ExtractErr(); err != nil {
		return "", fmt.Errorf("uploading image data: %w", err)
	}

	return newImage.ID, nil
}

func (u *Uploader) ensureImageDeleted(ctx context.Context) error {
	imageClient, err := u.image(ctx)
	if err != nil {
		return err
	}

	listOpts := images.ListOpts{
		Name:  u.config.OpenStack.ImageName,
		Limit: 1,
	}
	page, err := images.List(imageClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("listing images: %w", err)
	}
	imgs, err := images.ExtractImages(page)
	if err != nil {
		return fmt.Errorf("extracting images: %w", err)
	}
	if len(imgs) == 0 {
		return nil
	}
	if len(imgs) != 1 {
		return errors.New("multiple images with the same name found")
	}
	u.log.Printf("Deleting existing image %q (%s)", u.config.OpenStack.ImageName, imgs[0].ID)
	return images.Delete(imageClient, imgs[0].ID).ExtractErr()
}
