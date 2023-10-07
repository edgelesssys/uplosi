/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/

package gcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"path"
	"strings"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/storage"
	"github.com/edgelesssys/uplosi/config"
)

// Uploader can upload and remove os images on GCP.
type Uploader struct {
	config config.Config

	image  func(context.Context) (imagesAPI, error)
	bucket func(context.Context) (bucketAPI, error)

	log *log.Logger
}

// NewUploader creates a new config.
func NewUploader(config config.Config, log *log.Logger) (*Uploader, error) {
	return &Uploader{
		config: config,
		image: func(ctx context.Context) (imagesAPI, error) {
			return compute.NewImagesRESTClient(ctx)
		},
		bucket: func(ctx context.Context) (bucketAPI, error) {
			storage, err := storage.NewClient(ctx)
			if err != nil {
				return nil, err
			}
			return storage.Bucket(config.GCP.Bucket), nil
		},
		log: log,
	}, nil
}

// Upload uploads an OS image to GCP.
func (u *Uploader) Upload(ctx context.Context, image io.ReadSeeker, _ int64) (ref []string, retErr error) {
	// Ensure new image can be uploaded by deleting existing resources with the same name.
	if err := u.ensureImageDeleted(ctx); err != nil {
		return nil, fmt.Errorf("pre-cleaning: ensuring no image using the same name exists: %w", err)
	}
	if err := u.ensureBlobDeleted(ctx); err != nil {
		return nil, fmt.Errorf("pre-cleaning: ensuring no blob using the same name exists: %w", err)
	}

	// Ensure bucket exists.
	if err := u.ensureBucket(ctx); err != nil {
		return nil, fmt.Errorf("ensuring bucket exists: %w", err)
	}

	// Upload tar.gz encoded raw image to GCS.
	if err := u.uploadBlob(ctx, image); err != nil {
		return nil, fmt.Errorf("uploading image to GCS: %w", err)
	}
	defer func(retErr *error) {
		if err := u.ensureBlobDeleted(ctx); err != nil {
			*retErr = errors.Join(*retErr, fmt.Errorf("post-cleaning: deleting temporary blob from GCS: %w", err))
		}
	}(&retErr)

	imageRef, err := u.createImage(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating image: %w", err)
	}

	return []string{imageRef}, nil
}

func (u *Uploader) createImage(ctx context.Context) (string, error) {
	imageName := u.config.GCP.ImageName
	imageC, err := u.image(ctx)
	if err != nil {
		return "", err
	}

	u.log.Printf("Creating image %s", imageName)
	blobURL := blobURL(u.config.GCP.Bucket, u.config.GCP.BlobName)
	req := computepb.InsertImageRequest{
		ImageResource: &computepb.Image{
			Name: &imageName,
			RawDisk: &computepb.RawDisk{
				ContainerType: toPtr("TAR"),
				Source:        &blobURL,
			},
			Family:       &u.config.GCP.ImageFamily,
			Architecture: toPtr("X86_64"),
			GuestOsFeatures: []*computepb.GuestOsFeature{
				{Type: toPtr("GVNIC")},
				{Type: toPtr("SEV_CAPABLE")},
				{Type: toPtr("SEV_SNP_CAPABLE")},
				{Type: toPtr("VIRTIO_SCSI_MULTIQUEUE")},
				{Type: toPtr("UEFI_COMPATIBLE")},
			},
			// TODO(malt3): enable secure boot support
			// ShieldedInstanceInitialState: nil,
		},
		Project: u.config.GCP.Project,
	}
	op, err := imageC.Insert(ctx, &req)
	if err != nil {
		return "", fmt.Errorf("creating image: %w", err)
	}
	if err := op.Wait(ctx); err != nil {
		return "", fmt.Errorf("waiting for image to be created: %w", err)
	}
	policy := &computepb.Policy{
		Bindings: []*computepb.Binding{
			{
				Role:    toPtr("roles/compute.imageUser"),
				Members: []string{"allAuthenticatedUsers"},
			},
		},
	}
	if _, err = imageC.SetIamPolicy(ctx, &computepb.SetIamPolicyImageRequest{
		Resource: imageName,
		Project:  u.config.GCP.Project,
		GlobalSetPolicyRequestResource: &computepb.GlobalSetPolicyRequest{
			Policy: policy,
		},
	}); err != nil {
		return "", fmt.Errorf("setting iam policy: %w", err)
	}
	image, err := imageC.Get(ctx, &computepb.GetImageRequest{
		Image:   imageName,
		Project: u.config.GCP.Project,
	})
	if err != nil {
		return "", fmt.Errorf("created image doesn't exist: %w", err)
	}
	return strings.TrimPrefix(image.GetSelfLink(), "https://www.googleapis.com/compute/v1/"), nil
}

func (u *Uploader) uploadBlob(ctx context.Context, img io.Reader) error {
	blobName := u.config.GCP.BlobName
	bucketC, err := u.bucket(ctx)
	if err != nil {
		return err
	}
	u.log.Printf("Uploading os image as temporary blob %s", blobName)

	writer := bucketC.Object(blobName).NewWriter(ctx)
	_, err = io.Copy(writer, img)
	if err != nil {
		return err
	}
	return writer.Close()
}

func (u *Uploader) ensureImageDeleted(ctx context.Context) error {
	imageC, err := u.image(ctx)
	if err != nil {
		return err
	}
	imageName := u.config.GCP.ImageName
	if err != nil {
		return fmt.Errorf("inferring image name: %w", err)
	}

	_, err = imageC.Get(ctx, &computepb.GetImageRequest{
		Image:   imageName,
		Project: u.config.GCP.Project,
	})
	if err != nil {
		u.log.Printf("Image %s doesn't exist. Nothing to clean up.", imageName)
		return nil
	}
	u.log.Printf("Deleting image %s", imageName)
	op, err := imageC.Delete(ctx, &computepb.DeleteImageRequest{
		Image:   imageName,
		Project: u.config.GCP.Project,
	})
	if err != nil {
		return err
	}
	return op.Wait(ctx)
}

func (u *Uploader) ensureBlobDeleted(ctx context.Context) error {
	bucketC, err := u.bucket(ctx)
	if err != nil {
		return err
	}
	blobName := u.config.GCP.BlobName

	bucketExists, err := u.bucketExists(ctx)
	if err != nil {
		return err
	}
	if !bucketExists {
		u.log.Printf("Bucket %s doesn't exist. Nothing to clean up.", u.config.GCP.Bucket)
		return nil
	}

	_, err = bucketC.Object(blobName).Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		u.log.Printf("Blob %s doesn't exist. Nothing to clean up.", blobName)
		return nil
	}
	if err != nil {
		return err
	}
	u.log.Printf("Deleting blob %s", blobName)
	return bucketC.Object(blobName).Delete(ctx)
}

func (u *Uploader) ensureBucket(ctx context.Context) error {
	bucketC, err := u.bucket(ctx)
	if err != nil {
		return err
	}
	bucket := u.config.GCP.Bucket
	bucketExists, err := u.bucketExists(ctx)
	if err != nil {
		return err
	}
	if bucketExists {
		u.log.Printf("Bucket %s exists", bucket)
		return nil
	}
	u.log.Printf("Creating bucket %s", bucket)
	return bucketC.Create(ctx, u.config.GCP.Project, &storage.BucketAttrs{
		PublicAccessPrevention: storage.PublicAccessPreventionEnforced,
		Location:               u.config.GCP.Location,
	})
}

func (u *Uploader) bucketExists(ctx context.Context) (bool, error) {
	bucketC, err := u.bucket(ctx)
	if err != nil {
		return false, err
	}
	_, err = bucketC.Attrs(ctx)
	if err == nil {
		return true, err
	}
	if err == storage.ErrBucketNotExist {
		return false, nil
	}

	return false, err
}

func blobURL(bucketName, blobName string) string {
	return (&url.URL{
		Scheme: "https",
		Host:   "storage.googleapis.com",
		Path:   path.Join(bucketName, blobName),
	}).String()
}

func toPtr[T any](v T) *T {
	return &v
}
