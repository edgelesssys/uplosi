/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/
package aws

import (
	"context"

	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type ec2API interface {
	DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput,
		optFns ...func(*ec2.Options),
	) (*ec2.DescribeImagesOutput, error)
	ModifyImageAttribute(ctx context.Context, params *ec2.ModifyImageAttributeInput,
		optFns ...func(*ec2.Options),
	) (*ec2.ModifyImageAttributeOutput, error)
	RegisterImage(ctx context.Context, params *ec2.RegisterImageInput,
		optFns ...func(*ec2.Options),
	) (*ec2.RegisterImageOutput, error)
	CopyImage(ctx context.Context, params *ec2.CopyImageInput, optFns ...func(*ec2.Options),
	) (*ec2.CopyImageOutput, error)
	DeregisterImage(ctx context.Context, params *ec2.DeregisterImageInput,
		optFns ...func(*ec2.Options),
	) (*ec2.DeregisterImageOutput, error)
	ImportSnapshot(ctx context.Context, params *ec2.ImportSnapshotInput,
		optFns ...func(*ec2.Options),
	) (*ec2.ImportSnapshotOutput, error)
	DescribeImportSnapshotTasks(ctx context.Context, params *ec2.DescribeImportSnapshotTasksInput,
		optFns ...func(*ec2.Options),
	) (*ec2.DescribeImportSnapshotTasksOutput, error)
	DescribeSnapshots(ctx context.Context, params *ec2.DescribeSnapshotsInput,
		optFns ...func(*ec2.Options),
	) (*ec2.DescribeSnapshotsOutput, error)
	DeleteSnapshot(ctx context.Context, params *ec2.DeleteSnapshotInput, optFns ...func(*ec2.Options),
	) (*ec2.DeleteSnapshotOutput, error)
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options),
	) (*ec2.CreateTagsOutput, error)
}

type s3API interface {
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options),
	) (*s3.HeadBucketOutput, error)
	CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options),
	) (*s3.CreateBucketOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options),
	) (*s3.HeadObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options),
	) (*s3.DeleteObjectOutput, error)
}

type s3UploaderAPI interface {
	Upload(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3manager.Uploader),
	) (*s3manager.UploadOutput, error)
}

type stsAPI interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput,
		optFns ...func(*sts.Options),
	) (*sts.GetCallerIdentityOutput, error)
}
