/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/

package azure

import (
	"context"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	armcomputev5 "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/pageblob"
)

type sasBlobUploader func(sasBlobURL string) (azurePageblobAPI, error)

type azureGroupsAPI interface {
	CheckExistence(ctx context.Context, resourceGroupName string,
		options *armresources.ResourceGroupsClientCheckExistenceOptions,
	) (armresources.ResourceGroupsClientCheckExistenceResponse, error)
	Get(ctx context.Context, resourceGroupName string,
		options *armresources.ResourceGroupsClientGetOptions,
	) (armresources.ResourceGroupsClientGetResponse, error)
	CreateOrUpdate(ctx context.Context, resourceGroupName string, parameters armresources.ResourceGroup,
		options *armresources.ResourceGroupsClientCreateOrUpdateOptions,
	) (armresources.ResourceGroupsClientCreateOrUpdateResponse, error)
}

type azureDiskAPI interface {
	Get(ctx context.Context, resourceGroupName string, diskName string,
		options *armcomputev5.DisksClientGetOptions,
	) (armcomputev5.DisksClientGetResponse, error)
	BeginCreateOrUpdate(ctx context.Context, resourceGroupName string, diskName string, disk armcomputev5.Disk,
		options *armcomputev5.DisksClientBeginCreateOrUpdateOptions,
	) (*runtime.Poller[armcomputev5.DisksClientCreateOrUpdateResponse], error)
	BeginDelete(ctx context.Context, resourceGroupName string, diskName string,
		options *armcomputev5.DisksClientBeginDeleteOptions,
	) (*runtime.Poller[armcomputev5.DisksClientDeleteResponse], error)
	BeginGrantAccess(ctx context.Context, resourceGroupName string, diskName string, grantAccessData armcomputev5.GrantAccessData,
		options *armcomputev5.DisksClientBeginGrantAccessOptions,
	) (*runtime.Poller[armcomputev5.DisksClientGrantAccessResponse], error)
	BeginRevokeAccess(ctx context.Context, resourceGroupName string, diskName string,
		options *armcomputev5.DisksClientBeginRevokeAccessOptions,
	) (*runtime.Poller[armcomputev5.DisksClientRevokeAccessResponse], error)
}

type azureManagedImageAPI interface {
	Get(ctx context.Context, resourceGroupName string, imageName string,
		options *armcomputev5.ImagesClientGetOptions,
	) (armcomputev5.ImagesClientGetResponse, error)
	BeginCreateOrUpdate(ctx context.Context, resourceGroupName string,
		imageName string, parameters armcomputev5.Image,
		options *armcomputev5.ImagesClientBeginCreateOrUpdateOptions,
	) (*runtime.Poller[armcomputev5.ImagesClientCreateOrUpdateResponse], error)
	BeginDelete(ctx context.Context, resourceGroupName string, imageName string,
		options *armcomputev5.ImagesClientBeginDeleteOptions,
	) (*runtime.Poller[armcomputev5.ImagesClientDeleteResponse], error)
}

type azurePageblobAPI interface {
	UploadPages(ctx context.Context, body io.ReadSeekCloser, contentRange blob.HTTPRange,
		options *pageblob.UploadPagesOptions,
	) (pageblob.UploadPagesResponse, error)
}

type azureGalleriesAPI interface {
	Get(ctx context.Context, resourceGroupName string, galleryName string,
		options *armcomputev5.GalleriesClientGetOptions,
	) (armcomputev5.GalleriesClientGetResponse, error)
	NewListPager(options *armcomputev5.GalleriesClientListOptions,
	) *runtime.Pager[armcomputev5.GalleriesClientListResponse]
	BeginCreateOrUpdate(ctx context.Context, resourceGroupName string,
		galleryName string, gallery armcomputev5.Gallery,
		options *armcomputev5.GalleriesClientBeginCreateOrUpdateOptions,
	) (*runtime.Poller[armcomputev5.GalleriesClientCreateOrUpdateResponse], error)
}

type azureGalleriesImageAPI interface {
	Get(ctx context.Context, resourceGroupName string, galleryName string,
		galleryImageName string, options *armcomputev5.GalleryImagesClientGetOptions,
	) (armcomputev5.GalleryImagesClientGetResponse, error)
	BeginCreateOrUpdate(ctx context.Context, resourceGroupName string, galleryName string,
		galleryImageName string, galleryImage armcomputev5.GalleryImage,
		options *armcomputev5.GalleryImagesClientBeginCreateOrUpdateOptions,
	) (*runtime.Poller[armcomputev5.GalleryImagesClientCreateOrUpdateResponse], error)
	BeginDelete(ctx context.Context, resourceGroupName string, galleryName string, galleryImageName string,
		options *armcomputev5.GalleryImagesClientBeginDeleteOptions,
	) (*runtime.Poller[armcomputev5.GalleryImagesClientDeleteResponse], error)
}

type azureGalleriesImageVersionAPI interface {
	Get(ctx context.Context, resourceGroupName string, galleryName string, galleryImageName string, galleryImageVersionName string,
		options *armcomputev5.GalleryImageVersionsClientGetOptions,
	) (armcomputev5.GalleryImageVersionsClientGetResponse, error)
	NewListByGalleryImagePager(resourceGroupName string, galleryName string, galleryImageName string,
		options *armcomputev5.GalleryImageVersionsClientListByGalleryImageOptions,
	) *runtime.Pager[armcomputev5.GalleryImageVersionsClientListByGalleryImageResponse]
	BeginCreateOrUpdate(ctx context.Context, resourceGroupName string, galleryName string, galleryImageName string,
		galleryImageVersionName string, galleryImageVersion armcomputev5.GalleryImageVersion,
		options *armcomputev5.GalleryImageVersionsClientBeginCreateOrUpdateOptions,
	) (*runtime.Poller[armcomputev5.GalleryImageVersionsClientCreateOrUpdateResponse], error)
	BeginDelete(ctx context.Context, resourceGroupName string, galleryName string, galleryImageName string,
		galleryImageVersionName string, options *armcomputev5.GalleryImageVersionsClientBeginDeleteOptions,
	) (*runtime.Poller[armcomputev5.GalleryImageVersionsClientDeleteResponse], error)
}

type azureCommunityGalleryImageVersionAPI interface {
	Get(ctx context.Context, location string,
		publicGalleryName, galleryImageName, galleryImageVersionName string,
		options *armcomputev5.CommunityGalleryImageVersionsClientGetOptions,
	) (armcomputev5.CommunityGalleryImageVersionsClientGetResponse, error)
}

type azureGallerySharingProfileAPI interface {
	BeginUpdate(ctx context.Context, resourceGroupName string, galleryName string,
		sharingUpdate armcomputev5.SharingUpdate, options *armcomputev5.GallerySharingProfileClientBeginUpdateOptions,
	) (*runtime.Poller[armcomputev5.GallerySharingProfileClientUpdateResponse], error)
}
