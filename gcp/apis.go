/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/
package gcp

import (
	"context"
	"io"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/storage"
	gaxv2 "github.com/googleapis/gax-go/v2"
)

type imagesAPI interface {
	Get(ctx context.Context, req *computepb.GetImageRequest, opts ...gaxv2.CallOption,
	) (*computepb.Image, error)
	Insert(ctx context.Context, req *computepb.InsertImageRequest, opts ...gaxv2.CallOption,
	) (*compute.Operation, error)
	SetIamPolicy(ctx context.Context, req *computepb.SetIamPolicyImageRequest, opts ...gaxv2.CallOption,
	) (*computepb.Policy, error)
	Delete(ctx context.Context, req *computepb.DeleteImageRequest, opts ...gaxv2.CallOption,
	) (*compute.Operation, error)
	io.Closer
}

type bucketAPI interface {
	Attrs(ctx context.Context) (attrs *storage.BucketAttrs, err error)
	Create(ctx context.Context, projectID string, attrs *storage.BucketAttrs) (err error)
	Object(name string) *storage.ObjectHandle
}
