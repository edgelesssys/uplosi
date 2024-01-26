/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/

package azure

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	armcomputev5 "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/pageblob"
	"github.com/edgelesssys/uplosi/config"
)

const (
	pollingFrequency     = 10 * time.Second
	uploadAccessDuration = 86400   // 24 hours
	pageSizeMax          = 4194304 // 4MiB
	pageSizeMin          = 512     // 512 bytes
)

// Uploader can upload and remove os images on Azure.
type Uploader struct {
	config           config.Config
	pollingFrequency time.Duration
	pollOpts         *runtime.PollUntilDoneOptions

	disks             azureDiskAPI
	managedImages     azureManagedImageAPI
	blob              sasBlobUploader
	galleries         azureGalleriesAPI
	image             azureGalleriesImageAPI
	imageVersions     azureGalleriesImageVersionAPI
	communityVersions azureCommunityGalleryImageVersionAPI
	gallerySharing    azureGallerySharingProfileAPI

	log *log.Logger
}

// NewUploader creates a new config.
func NewUploader(config config.Config, log *log.Logger) (*Uploader, error) {
	subscriptionID := config.Azure.SubscriptionID

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}
	diskClient, err := armcomputev5.NewDisksClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}
	managedImagesClient, err := armcomputev5.NewImagesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}
	galleriesClient, err := armcomputev5.NewGalleriesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}
	galleriesImageClient, err := armcomputev5.NewGalleryImagesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}
	galleriesImageVersionClient, err := armcomputev5.NewGalleryImageVersionsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}
	communityImageVersionClient, err := armcomputev5.NewCommunityGalleryImageVersionsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}
	gallerySharingClient, err := armcomputev5.NewGallerySharingProfileClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	return &Uploader{
		config:           config,
		pollingFrequency: pollingFrequency,
		pollOpts:         &runtime.PollUntilDoneOptions{Frequency: pollingFrequency},
		disks:            diskClient,
		managedImages:    managedImagesClient,
		blob: func(sasBlobURL string) (azurePageblobAPI, error) {
			return pageblob.NewClientWithNoCredential(sasBlobURL, nil)
		},
		galleries:         galleriesClient,
		image:             galleriesImageClient,
		imageVersions:     galleriesImageVersionClient,
		communityVersions: communityImageVersionClient,
		gallerySharing:    gallerySharingClient,
		log:               log,
	}, nil
}

// Upload uploads an OS image to Azure.
func (u *Uploader) Upload(ctx context.Context, image io.ReadSeeker, size int64) (refs []string, retErr error) {
	// Ensure new image can be uploaded by deleting existing resources using the same name.
	if err := u.ensureImageVersionDeleted(ctx); err != nil {
		return nil, fmt.Errorf("pre-cleaning: ensuring no image version using the same name exists: %w", err)
	}
	if err := u.ensureManagedImageDeleted(ctx); err != nil {
		return nil, fmt.Errorf("pre-cleaning: ensuring no managed image using the same name exists: %w", err)
	}
	if err := u.ensureDiskDeleted(ctx); err != nil {
		return nil, fmt.Errorf("pre-cleaning: ensuring no temporary disk using the same name exists: %w", err)
	}

	// Ensure SIG and image definition exist.
	// These aren't cleaned up as they are shared between images.
	if err := u.ensureSIG(ctx); err != nil {
		return nil, fmt.Errorf("ensuring sig exists: %w", err)
	}
	if err := u.ensureImageDefinition(ctx); err != nil {
		return nil, fmt.Errorf("ensuring image definition exists: %w", err)
	}

	vhdReader := newVHDReader(image, uint64(size), [16]byte{}, time.Time{})
	diskID, err := u.createDisk(ctx, DiskTypeNormal, vhdReader, nil, int64(vhdReader.ContainerSize()))
	if err != nil {
		return nil, fmt.Errorf("creating disk: %w", err)
	}
	defer func(retErr *error) {
		// cleanup temp disk
		if err := u.ensureDiskDeleted(ctx); err != nil {
			*retErr = errors.Join(*retErr, fmt.Errorf("post-cleaning: deleting disk image: %v", err))
		}
	}(&retErr)

	managedImageID, err := u.createManagedImage(ctx, diskID)
	if err != nil {
		return nil, fmt.Errorf("creating managed image: %w", err)
	}
	unsharedImageVersionID, err := u.createImageVersion(ctx, managedImageID)
	if err != nil {
		return nil, fmt.Errorf("creating image version: %w", err)
	}

	imageReference, err := u.getImageReference(ctx, unsharedImageVersionID)
	if err != nil {
		return nil, fmt.Errorf("getting image reference: %w", err)
	}

	return []string{imageReference}, nil
}

// createDisk creates and initializes (uploads contents of) an azure disk.
func (u *Uploader) createDisk(ctx context.Context, diskType DiskType, img io.Reader, vmgs io.ReadSeeker, size int64) (string, error) {
	rg := u.config.Azure.ResourceGroup
	diskName := u.config.Azure.DiskName

	u.log.Printf("Creating disk %s in %s", diskName, rg)
	if diskType == DiskTypeWithVMGS && vmgs == nil {
		return "", errors.New("cannot create disk with vmgs: vmgs reader is nil")
	}
	var createOption armcomputev5.DiskCreateOption
	var requestVMGSSAS bool
	switch diskType {
	case DiskTypeNormal:
		createOption = armcomputev5.DiskCreateOptionUpload
	case DiskTypeWithVMGS:
		createOption = armcomputev5.DiskCreateOptionUploadPreparedSecure
		requestVMGSSAS = true
	}

	disk := armcomputev5.Disk{
		Location: &u.config.Azure.Location,
		Properties: &armcomputev5.DiskProperties{
			CreationData: &armcomputev5.CreationData{
				CreateOption:    &createOption,
				UploadSizeBytes: toPtr(size),
			},
			HyperVGeneration: toPtr(armcomputev5.HyperVGenerationV2),
			OSType:           toPtr(armcomputev5.OperatingSystemTypesLinux),
		},
	}
	createPoller, err := u.disks.BeginCreateOrUpdate(ctx, rg, diskName, disk, &armcomputev5.DisksClientBeginCreateOrUpdateOptions{})
	if err != nil {
		return "", fmt.Errorf("creating disk: %w", err)
	}
	createdDisk, err := createPoller.PollUntilDone(ctx, u.pollOpts)
	if err != nil {
		return "", fmt.Errorf("waiting for disk to be created: %w", err)
	}

	u.log.Printf("Granting temporary upload permissions via SAS token")
	accessGrant := armcomputev5.GrantAccessData{
		Access:                   toPtr(armcomputev5.AccessLevelWrite),
		DurationInSeconds:        toPtr(int32(uploadAccessDuration)),
		GetSecureVMGuestStateSAS: &requestVMGSSAS,
	}
	accessPoller, err := u.disks.BeginGrantAccess(ctx, rg, diskName, accessGrant, &armcomputev5.DisksClientBeginGrantAccessOptions{})
	if err != nil {
		return "", fmt.Errorf("generating disk sas token: %w", err)
	}
	accesPollerResp, err := accessPoller.PollUntilDone(ctx, u.pollOpts)
	if err != nil {
		return "", fmt.Errorf("waiting for sas token: %w", err)
	}

	if requestVMGSSAS {
		u.log.Printf("Uploading vmgs")
		vmgsSize, err := vmgs.Seek(0, io.SeekEnd)
		if err != nil {
			return "", err
		}
		if _, err := vmgs.Seek(0, io.SeekStart); err != nil {
			return "", err
		}
		if accesPollerResp.SecurityDataAccessSAS == nil {
			return "", errors.New("uploading vmgs: grant access returned no vmgs sas")
		}
		if err := uploadBlob(ctx, *accesPollerResp.SecurityDataAccessSAS, vmgs, vmgsSize, u.blob); err != nil {
			return "", fmt.Errorf("uploading vmgs: %w", err)
		}
	}

	u.log.Printf("Uploading os image")
	if accesPollerResp.AccessSAS == nil {
		return "", errors.New("uploading disk: grant access returned no disk sas")
	}
	if err := uploadBlob(ctx, *accesPollerResp.AccessSAS, img, size, u.blob); err != nil {
		return "", fmt.Errorf("uploading image: %w", err)
	}

	revokePoller, err := u.disks.BeginRevokeAccess(ctx, rg, diskName, &armcomputev5.DisksClientBeginRevokeAccessOptions{})
	if err != nil {
		return "", fmt.Errorf("revoking disk sas token: %w", err)
	}
	if _, err := revokePoller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{Frequency: u.pollingFrequency}); err != nil {
		return "", fmt.Errorf("waiting for sas token revocation: %w", err)
	}

	if createdDisk.ID == nil {
		return "", errors.New("created disk has no id")
	}

	return *createdDisk.ID, nil
}

func (u *Uploader) ensureDiskDeleted(ctx context.Context) error {
	rg := u.config.Azure.ResourceGroup
	diskName := u.config.Azure.DiskName

	getOpts := &armcomputev5.DisksClientGetOptions{}
	if _, err := u.disks.Get(ctx, rg, diskName, getOpts); err != nil {
		u.log.Printf("Disk %s in %s doesn't exist. Nothing to clean up.", diskName, rg)
		return nil
	}

	u.log.Printf("Deleting disk %s in %s", diskName, rg)
	deleteOpts := &armcomputev5.DisksClientBeginDeleteOptions{}
	deletePoller, err := u.disks.BeginDelete(ctx, rg, diskName, deleteOpts)
	if err != nil {
		return fmt.Errorf("deleting disk: %w", err)
	}

	if _, err = deletePoller.PollUntilDone(ctx, u.pollOpts); err != nil {
		return fmt.Errorf("waiting for disk to be deleted: %w", err)
	}
	return nil
}

func (u *Uploader) createManagedImage(ctx context.Context, diskID string) (string, error) {
	rg := u.config.Azure.ResourceGroup
	location := u.config.Azure.Location
	imgName := u.config.Azure.DiskName

	u.log.Printf("Creating managed image %s in %s", imgName, rg)
	image := armcomputev5.Image{
		Location: &location,
		Properties: &armcomputev5.ImageProperties{
			HyperVGeneration: toPtr(armcomputev5.HyperVGenerationTypesV2),
			StorageProfile: &armcomputev5.ImageStorageProfile{
				OSDisk: &armcomputev5.ImageOSDisk{
					OSState: toPtr(armcomputev5.OperatingSystemStateTypesGeneralized),
					OSType:  toPtr(armcomputev5.OperatingSystemTypesLinux),
					ManagedDisk: &armcomputev5.SubResource{
						ID: &diskID,
					},
				},
			},
		},
	}
	opts := &armcomputev5.ImagesClientBeginCreateOrUpdateOptions{}
	createPoller, err := u.managedImages.BeginCreateOrUpdate(ctx, rg, imgName, image, opts)
	if err != nil {
		return "", fmt.Errorf("creating managed image: %w", err)
	}
	createdImage, err := createPoller.PollUntilDone(ctx, u.pollOpts)
	if err != nil {
		return "", fmt.Errorf("waiting for image to be created: %w", err)
	}

	if createdImage.ID == nil {
		return "", errors.New("created image has no id")
	}

	return *createdImage.ID, nil
}

func (u *Uploader) ensureManagedImageDeleted(ctx context.Context) error {
	rg := u.config.Azure.ResourceGroup
	imgName := u.config.Azure.DiskName

	getOpts := &armcomputev5.ImagesClientGetOptions{}
	if _, err := u.managedImages.Get(ctx, rg, imgName, getOpts); err != nil {
		u.log.Printf("Managed image %s in %s doesn't exist. Nothing to clean up.", imgName, rg)
		return nil
	}

	u.log.Printf("Deleting managed image %s in %s", imgName, rg)
	deleteOpts := &armcomputev5.ImagesClientBeginDeleteOptions{}
	deletePoller, err := u.managedImages.BeginDelete(ctx, rg, imgName, deleteOpts)
	if err != nil {
		return fmt.Errorf("deleting image: %w", err)
	}

	if _, err = deletePoller.PollUntilDone(ctx, u.pollOpts); err != nil {
		return fmt.Errorf("waiting for image to be deleted: %w", err)
	}
	return nil
}

// ensureSIG creates a SIG if it does not exist yet.
func (u *Uploader) ensureSIG(ctx context.Context) error {
	rg := u.config.Azure.ResourceGroup
	sigName := u.config.Azure.SharedImageGallery
	pubNamePrefix := u.config.Azure.SharingNamePrefix
	sharingProf := sharingProfilePermissionFromString(u.config.Azure.SharingProfile)

	resp, err := u.galleries.Get(ctx, rg, sigName, &armcomputev5.GalleriesClientGetOptions{})
	if err == nil {
		u.log.Printf("Image gallery %s in %s exists", sigName, rg)
		if resp.Gallery.Properties == nil {
			return errors.New("image gallery has no properties")
		}
		if resp.Gallery.Properties.SharingProfile == nil {
			return errors.New("image gallery has no sharing profile")
		}
		if resp.Gallery.Properties.SharingProfile.Permissions == nil {
			return errors.New("image gallery has no sharing profile permissions")
		}
		if *resp.Gallery.Properties.SharingProfile.Permissions != *sharingProf {
			return errors.New("image gallery has different sharing profile permissions, cannot update automatically")
		}
		return nil
	}
	u.log.Printf("Creating image gallery %s in %s", sigName, rg)
	gallery := armcomputev5.Gallery{
		Location: &u.config.Azure.Location,
		Properties: &armcomputev5.GalleryProperties{
			SharingProfile: &armcomputev5.SharingProfile{
				CommunityGalleryInfo: &armcomputev5.CommunityGalleryInfo{
					PublicNamePrefix: &pubNamePrefix,
					Eula:             toPtr("none"),
					PublisherContact: toPtr("test@foo.bar"),
					PublisherURI:     toPtr("https://foo.bar"),
				},
				Permissions: sharingProf,
			},
		},
	}
	opts := &armcomputev5.GalleriesClientBeginCreateOrUpdateOptions{}
	createPoller, err := u.galleries.BeginCreateOrUpdate(ctx, rg, sigName, gallery, opts)
	if err != nil {
		return fmt.Errorf("creating image gallery: %w", err)
	}
	if _, err = createPoller.PollUntilDone(ctx, u.pollOpts); err != nil {
		return fmt.Errorf("waiting for image gallery to be created: %w", err)
	}

	if u.config.Azure.SharingProfile == "community" {
		sharingUpdate := armcomputev5.SharingUpdate{
			OperationType: toPtr(armcomputev5.SharingUpdateOperationTypesEnableCommunity),
		}
		if _, err := u.gallerySharing.BeginUpdate(ctx, rg, sigName, sharingUpdate, nil); err != nil {
			return fmt.Errorf("enabling community sharing: %w", err)
		}
	}

	return nil
}

func sharingProfilePermissionFromString(s string) *armcomputev5.GallerySharingPermissionTypes {
	switch strings.ToLower(s) {
	case "community":
		return toPtr(armcomputev5.GallerySharingPermissionTypesCommunity)
	// case "groups":
	// 	return armcomputev5.GallerySharingPermissionTypesGroups
	default:
		return toPtr(armcomputev5.GallerySharingPermissionTypesPrivate)
	}
}

// ensureImageDefinition creates an image definition (component of a SIG) if it does not exist yet.
func (u *Uploader) ensureImageDefinition(ctx context.Context) error {
	rg := u.config.Azure.ResourceGroup
	sigName := u.config.Azure.SharedImageGallery
	attestVariant := u.config.Azure.AttestationVariant
	defName := u.config.Azure.ImageDefinitionName

	_, err := u.image.Get(ctx, rg, sigName, defName, &armcomputev5.GalleryImagesClientGetOptions{})
	if err == nil {
		u.log.Printf("Image definition %s/%s in %s exists", sigName, defName, rg)
		return nil
	}
	u.log.Printf("Creating image definition  %s/%s in %s", sigName, defName, rg)
	var securityType string
	// TODO(malt3): This needs to allow the *Supported or the normal variant
	// based on wether a VMGS was provided or not.
	// VMGS provided: ConfidentialVM
	// No VMGS provided: ConfidentialVMSupported
	switch strings.ToLower(attestVariant) {
	case "azure-sev-snp", "azure-tdx":
		securityType = string("ConfidentialVMSupported")
	case "azure-trustedlaunch":
		securityType = string(armcomputev5.SecurityTypesTrustedLaunch)
	}

	galleryImage := armcomputev5.GalleryImage{
		Location: &u.config.Azure.Location,
		Properties: &armcomputev5.GalleryImageProperties{
			Identifier: &armcomputev5.GalleryImageIdentifier{
				Offer:     &u.config.Azure.Offer,
				Publisher: &u.config.Azure.Publisher,
				SKU:       &u.config.Azure.SKU,
			},
			OSState:      toPtr(armcomputev5.OperatingSystemStateTypesGeneralized),
			OSType:       toPtr(armcomputev5.OperatingSystemTypesLinux),
			Architecture: toPtr(armcomputev5.ArchitectureX64),
			Features: []*armcomputev5.GalleryImageFeature{
				{Name: toPtr("SecurityType"), Value: &securityType},
			},
			HyperVGeneration: toPtr(armcomputev5.HyperVGenerationV2),
		},
	}
	opts := &armcomputev5.GalleryImagesClientBeginCreateOrUpdateOptions{}
	createPoller, err := u.image.BeginCreateOrUpdate(ctx, rg, sigName, defName, galleryImage, opts)
	if err != nil {
		return fmt.Errorf("creating image definition: %w", err)
	}
	if _, err = createPoller.PollUntilDone(ctx, u.pollOpts); err != nil {
		return fmt.Errorf("waiting for image definition to be created: %w", err)
	}

	return nil
}

func (u *Uploader) createImageVersion(ctx context.Context, imageID string) (string, error) {
	rg := u.config.Azure.ResourceGroup
	sigName := u.config.Azure.SharedImageGallery
	verName := u.config.ImageVersion
	defName := u.config.Azure.ImageDefinitionName

	u.log.Printf("Creating image version %s/%s/%s in %s", sigName, defName, verName, rg)
	imageVersion := armcomputev5.GalleryImageVersion{
		Location: &u.config.Azure.Location,
		Properties: &armcomputev5.GalleryImageVersionProperties{
			StorageProfile: &armcomputev5.GalleryImageVersionStorageProfile{
				OSDiskImage: &armcomputev5.GalleryOSDiskImage{
					HostCaching: toPtr(armcomputev5.HostCachingReadOnly),
				},
				Source: &armcomputev5.GalleryArtifactVersionFullSource{
					ID: &imageID,
				},
			},
			PublishingProfile: &armcomputev5.GalleryImageVersionPublishingProfile{
				ReplicaCount:    toPtr[int32](1),
				ReplicationMode: toPtr(armcomputev5.ReplicationModeFull),
				TargetRegions:   replication(u.config.Azure.Location, u.config.Azure.ReplicationRegions, 1),
			},
		},
	}
	createPoller, err := u.imageVersions.BeginCreateOrUpdate(ctx, rg, sigName, defName, verName, imageVersion,
		&armcomputev5.GalleryImageVersionsClientBeginCreateOrUpdateOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("creating image version: %w", err)
	}
	createdImage, err := createPoller.PollUntilDone(ctx, u.pollOpts)
	if err != nil {
		return "", fmt.Errorf("waiting for image version to be created: %w", err)
	}
	if createdImage.ID == nil {
		return "", errors.New("created image has no id")
	}
	return *createdImage.ID, nil
}

func (u *Uploader) ensureImageVersionDeleted(ctx context.Context) error {
	rg := u.config.Azure.ResourceGroup
	sigName := u.config.Azure.SharedImageGallery
	verName := u.config.ImageVersion
	defName := u.config.Azure.ImageDefinitionName

	getOpts := &armcomputev5.GalleryImageVersionsClientGetOptions{}
	if _, err := u.imageVersions.Get(ctx, rg, sigName, defName, verName, getOpts); err != nil {
		u.log.Printf("Image version %s in %s/%s/%s doesn't exist. Nothing to clean up.", verName, rg, sigName, defName)
		return nil
	}

	u.log.Printf("Deleting image version %s in %s/%s/%s", verName, rg, sigName, defName)
	deleteOpts := &armcomputev5.GalleryImageVersionsClientBeginDeleteOptions{}
	deletePoller, err := u.imageVersions.BeginDelete(ctx, rg, sigName, defName, verName, deleteOpts)
	if err != nil {
		return fmt.Errorf("deleting image version: %w", err)
	}

	if _, err = deletePoller.PollUntilDone(ctx, u.pollOpts); err != nil {
		return fmt.Errorf("waiting for image version to be deleted: %w", err)
	}
	return nil
}

// getImageReference returns the image reference to use for the image version.
// If the shared image gallery is a community gallery, the community identifier is returned.
// Otherwise, the unshared identifier is returned.
func (u *Uploader) getImageReference(ctx context.Context, unsharedID string) (string, error) {
	rg := u.config.Azure.ResourceGroup
	location := u.config.Azure.Location
	sigName := u.config.Azure.SharedImageGallery
	verName := u.config.ImageVersion
	defName := u.config.Azure.ImageDefinitionName

	galleryResp, err := u.galleries.Get(ctx, rg, sigName, &armcomputev5.GalleriesClientGetOptions{})
	if err != nil {
		return "", fmt.Errorf("getting image gallery %s: %w", sigName, err)
	}
	if galleryResp.Properties == nil ||
		galleryResp.Properties.SharingProfile == nil ||
		galleryResp.Properties.SharingProfile.CommunityGalleryInfo == nil ||
		galleryResp.Properties.SharingProfile.CommunityGalleryInfo.CommunityGalleryEnabled == nil ||
		!*galleryResp.Properties.SharingProfile.CommunityGalleryInfo.CommunityGalleryEnabled {
		u.log.Printf("Image gallery %s in %s is not shared. Using private identifier", sigName, rg)
		return unsharedID, nil
	}
	if galleryResp.Properties == nil ||
		galleryResp.Properties.SharingProfile == nil ||
		galleryResp.Properties.SharingProfile.CommunityGalleryInfo == nil ||
		galleryResp.Properties.SharingProfile.CommunityGalleryInfo.PublicNames == nil ||
		len(galleryResp.Properties.SharingProfile.CommunityGalleryInfo.PublicNames) < 1 ||
		galleryResp.Properties.SharingProfile.CommunityGalleryInfo.PublicNames[0] == nil {
		return "", fmt.Errorf("image gallery %s in %s is a community gallery but has no public names", sigName, rg)
	}
	communityGalleryName := *galleryResp.Properties.SharingProfile.CommunityGalleryInfo.PublicNames[0]
	u.log.Printf("Image gallery %s in %s is shared. Using community identifier in %s", sigName, rg, communityGalleryName)
	opts := &armcomputev5.CommunityGalleryImageVersionsClientGetOptions{}
	communityVersionResp, err := u.communityVersions.Get(ctx, location, communityGalleryName, defName, verName, opts)
	if err != nil {
		return "", fmt.Errorf("getting community image version %s/%s/%s: %w", communityGalleryName, defName, verName, err)
	}
	if communityVersionResp.Identifier == nil || communityVersionResp.Identifier.UniqueID == nil {
		return "", fmt.Errorf("community image version %s/%s/%s has no id", communityGalleryName, defName, verName)
	}
	return *communityVersionResp.Identifier.UniqueID, nil
}

func uploadBlob(ctx context.Context, sasURL string, disk io.Reader, size int64, uploader sasBlobUploader) error {
	uploadClient, err := uploader(sasURL)
	if err != nil {
		return fmt.Errorf("uploading blob: %w", err)
	}
	var offset int64
	var chunksize int
	chunk := make([]byte, pageSizeMax)
	var readErr error
	for offset < size {
		chunksize, readErr = io.ReadAtLeast(disk, chunk, 1)
		if readErr != nil {
			return fmt.Errorf("reading from disk: %w", err)
		}
		if err := uploadChunk(ctx, uploadClient, bytes.NewReader(chunk[:chunksize]), offset, int64(chunksize)); err != nil {
			return fmt.Errorf("uploading chunk: %w", err)
		}
		offset += int64(chunksize)
	}
	return nil
}

func uploadChunk(ctx context.Context, uploader azurePageblobAPI, chunk io.ReadSeeker, offset, chunksize int64) error {
	_, err := uploader.UploadPages(ctx, &readSeekNopCloser{chunk}, blob.HTTPRange{
		Offset: offset,
		Count:  chunksize,
	}, nil)
	return err
}

type readSeekNopCloser struct {
	io.ReadSeeker
}

func (n *readSeekNopCloser) Close() error {
	return nil
}

func toPtr[T any](t T) *T {
	return &t
}

func replication(location string, regions []string, count int32) []*armcomputev5.TargetRegion {
	targetRegions := []*armcomputev5.TargetRegion{
		{
			Name:                 toPtr(location),
			RegionalReplicaCount: toPtr[int32](count),
		},
	}
	for _, region := range regions {
		if region == location {
			continue
		}
		targetRegions = append(targetRegions, &armcomputev5.TargetRegion{
			Name:                 toPtr(region),
			RegionalReplicaCount: toPtr[int32](count),
		})
	}

	return targetRegions
}
