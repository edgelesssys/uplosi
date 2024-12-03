/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/

package azure

import (
	"context"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	armcomputev6 "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
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
		options *armcomputev6.DisksClientGetOptions,
	) (armcomputev6.DisksClientGetResponse, error)
	BeginCreateOrUpdate(ctx context.Context, resourceGroupName string, diskName string, disk armcomputev6.Disk,
		options *armcomputev6.DisksClientBeginCreateOrUpdateOptions,
	) (*runtime.Poller[armcomputev6.DisksClientCreateOrUpdateResponse], error)
	BeginDelete(ctx context.Context, resourceGroupName string, diskName string,
		options *armcomputev6.DisksClientBeginDeleteOptions,
	) (*runtime.Poller[armcomputev6.DisksClientDeleteResponse], error)
	BeginGrantAccess(ctx context.Context, resourceGroupName string, diskName string, grantAccessData armcomputev6.GrantAccessData,
		options *armcomputev6.DisksClientBeginGrantAccessOptions,
	) (*runtime.Poller[armcomputev6.DisksClientGrantAccessResponse], error)
	BeginRevokeAccess(ctx context.Context, resourceGroupName string, diskName string,
		options *armcomputev6.DisksClientBeginRevokeAccessOptions,
	) (*runtime.Poller[armcomputev6.DisksClientRevokeAccessResponse], error)
}

type azureManagedImageAPI interface {
	Get(ctx context.Context, resourceGroupName string, imageName string,
		options *armcomputev6.ImagesClientGetOptions,
	) (armcomputev6.ImagesClientGetResponse, error)
	BeginCreateOrUpdate(ctx context.Context, resourceGroupName string,
		imageName string, parameters armcomputev6.Image,
		options *armcomputev6.ImagesClientBeginCreateOrUpdateOptions,
	) (*runtime.Poller[armcomputev6.ImagesClientCreateOrUpdateResponse], error)
	BeginDelete(ctx context.Context, resourceGroupName string, imageName string,
		options *armcomputev6.ImagesClientBeginDeleteOptions,
	) (*runtime.Poller[armcomputev6.ImagesClientDeleteResponse], error)
}

type azurePageblobAPI interface {
	UploadPages(ctx context.Context, body io.ReadSeekCloser, contentRange blob.HTTPRange,
		options *pageblob.UploadPagesOptions,
	) (pageblob.UploadPagesResponse, error)
}

type azureGalleriesAPI interface {
	Get(ctx context.Context, resourceGroupName string, galleryName string,
		options *armcomputev6.GalleriesClientGetOptions,
	) (armcomputev6.GalleriesClientGetResponse, error)
	NewListPager(options *armcomputev6.GalleriesClientListOptions,
	) *runtime.Pager[armcomputev6.GalleriesClientListResponse]
	BeginCreateOrUpdate(ctx context.Context, resourceGroupName string,
		galleryName string, gallery armcomputev6.Gallery,
		options *armcomputev6.GalleriesClientBeginCreateOrUpdateOptions,
	) (*runtime.Poller[armcomputev6.GalleriesClientCreateOrUpdateResponse], error)
}

type azureGalleriesImageAPI interface {
	Get(ctx context.Context, resourceGroupName string, galleryName string,
		galleryImageName string, options *armcomputev6.GalleryImagesClientGetOptions,
	) (armcomputev6.GalleryImagesClientGetResponse, error)
	BeginCreateOrUpdate(ctx context.Context, resourceGroupName string, galleryName string,
		galleryImageName string, galleryImage armcomputev6.GalleryImage,
		options *armcomputev6.GalleryImagesClientBeginCreateOrUpdateOptions,
	) (*runtime.Poller[armcomputev6.GalleryImagesClientCreateOrUpdateResponse], error)
	BeginDelete(ctx context.Context, resourceGroupName string, galleryName string, galleryImageName string,
		options *armcomputev6.GalleryImagesClientBeginDeleteOptions,
	) (*runtime.Poller[armcomputev6.GalleryImagesClientDeleteResponse], error)
}

type azureGalleriesImageVersionAPI interface {
	Get(ctx context.Context, resourceGroupName string, galleryName string, galleryImageName string, galleryImageVersionName string,
		options *armcomputev6.GalleryImageVersionsClientGetOptions,
	) (armcomputev6.GalleryImageVersionsClientGetResponse, error)
	NewListByGalleryImagePager(resourceGroupName string, galleryName string, galleryImageName string,
		options *armcomputev6.GalleryImageVersionsClientListByGalleryImageOptions,
	) *runtime.Pager[armcomputev6.GalleryImageVersionsClientListByGalleryImageResponse]
	BeginCreateOrUpdate(ctx context.Context, resourceGroupName string, galleryName string, galleryImageName string,
		galleryImageVersionName string, galleryImageVersion armcomputev6.GalleryImageVersion,
		options *armcomputev6.GalleryImageVersionsClientBeginCreateOrUpdateOptions,
	) (*runtime.Poller[armcomputev6.GalleryImageVersionsClientCreateOrUpdateResponse], error)
	BeginDelete(ctx context.Context, resourceGroupName string, galleryName string, galleryImageName string,
		galleryImageVersionName string, options *armcomputev6.GalleryImageVersionsClientBeginDeleteOptions,
	) (*runtime.Poller[armcomputev6.GalleryImageVersionsClientDeleteResponse], error)
}

type azureCommunityGalleryImageVersionAPI interface {
	Get(ctx context.Context, location string,
		publicGalleryName, galleryImageName, galleryImageVersionName string,
		options *armcomputev6.CommunityGalleryImageVersionsClientGetOptions,
	) (armcomputev6.CommunityGalleryImageVersionsClientGetResponse, error)
}

type azureGallerySharingProfileAPI interface {
	BeginUpdate(ctx context.Context, resourceGroupName string, galleryName string,
		sharingUpdate armcomputev6.SharingUpdate, options *armcomputev6.GallerySharingProfileClientBeginUpdateOptions,
	) (*runtime.Poller[armcomputev6.GallerySharingProfileClientUpdateResponse], error)
}
