/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/
package aws

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"text/template"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
	"github.com/edgelesssys/uplosi/uploader"
)

const (
	waitInterval = 15 * time.Second // 15 seconds
	maxWait      = 30 * time.Minute // 30 minutes
)

var errAMIDoesNotExist = errors.New("ami does not exist")

// Uploader can upload and remove os images on AWS.
type Uploader struct {
	config          uploader.Config
	amiNameTemplate *template.Template

	log *log.Logger
}

func NewUploader(config uploader.Config, log *log.Logger) (*Uploader, error) {
	templateString := config.AWS.AMINameTemplate
	if len(config.AWS.AMINameTemplate) == 0 {
		templateString = "{{.Name}}-{{.ImageVersion}}"
	}
	amiNameTemplate, err := template.New("ami-name").Parse(templateString)
	if err != nil {
		return nil, fmt.Errorf("parsing ami name template: %w", err)
	}

	return &Uploader{
		config:          config,
		amiNameTemplate: amiNameTemplate,
		log:             log,
	}, nil
}

func (u *Uploader) Upload(ctx context.Context, req *uploader.Request) (refs []string, retErr error) {
	allRegions := make([]string, 0, len(u.config.AWS.ReplicationRegions)+1)
	allRegions = append(allRegions, u.config.AWS.Region)
	allRegions = append(allRegions, u.config.AWS.ReplicationRegions...)
	amiIDs := make(map[string]string, len(allRegions))

	accountID, err := u.accountID(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting account ID: %w", err)
	}
	u.log.Printf("Uploading image to AWS account %s", accountID)

	// Ensure new image can be uploaded by deleting existing resources using the same name.
	for _, region := range allRegions {
		if err := u.ensureImageDeleted(ctx, region); err != nil {
			return nil, fmt.Errorf("pre-cleaning: ensuring no image under the name %s in region %s: %w", u.config.Name, region, err)
		}
	}
	if err := u.ensureSnapshotDeleted(ctx); err != nil {
		return nil, fmt.Errorf("pre-cleaning: ensuring no snapshot using the same name exists: %w", err)
	}
	if err := u.ensureBlobDeleted(ctx); err != nil {
		return nil, fmt.Errorf("pre-cleaning: ensuring no blob using the same name exists: %w", err)
	}

	// Ensure bucket exists.
	// While the blob is only created temporarily, the bucket is persistent.
	if err := u.ensureBucket(ctx); err != nil {
		return nil, fmt.Errorf("ensuring bucket exists: %w", err)
	}

	// create primary image
	if err := u.uploadBlob(ctx, req.Image); err != nil {
		return nil, fmt.Errorf("uploading image to s3: %w", err)
	}
	defer func(retErr *error) {
		if err := u.ensureBlobDeleted(ctx); err != nil {
			*retErr = errors.Join(*retErr, fmt.Errorf("post-cleaning: deleting temporary blob from s3: %w", err))
		}
	}(&retErr)
	snapshotID, err := u.importSnapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("importing snapshot: %w", err)
	}
	primaryAMIID, err := u.createImageFromSnapshot(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("creating image from snapshot: %w", err)
	}
	amiIDs[u.config.AWS.Region] = primaryAMIID
	if err := u.waitForImage(ctx, primaryAMIID, u.config.AWS.Region); err != nil {
		return nil, fmt.Errorf("waiting for primary image to become available: %w", err)
	}

	// replicate image
	for _, region := range u.config.AWS.ReplicationRegions {
		if _, alreadyReplicated := amiIDs[region]; alreadyReplicated {
			u.log.Printf("image was already replicated in region %s. Skipping.", region)
			continue
		}
		amiID, err := u.replicateImage(ctx, primaryAMIID, region)
		if err != nil {
			return nil, fmt.Errorf("replicating image to region %s: %w", region, err)
		}
		amiIDs[region] = amiID
	}

	// wait for replication, tag, publish
	// TODO(malt3): this has to collect a slice of (region, amiID) pairs and return them all
	amiARNs := make([]string, 0, len(allRegions))
	for _, region := range allRegions {
		if err := u.waitForImage(ctx, amiIDs[region], region); err != nil {
			return nil, fmt.Errorf("waiting for image to become available in region %s: %w", region, err)
		}
		if err := u.tagImageAndSnapshot(ctx, amiIDs[region], region); err != nil {
			return nil, fmt.Errorf("tagging image in region %s: %w", region, err)
		}
		if err := u.publishImage(ctx, amiIDs[region], region); err != nil {
			return nil, fmt.Errorf("publishing image in region %s: %w", region, err)
		}
		amiARNs = append(amiARNs, getAMIARN(accountID, region, amiIDs[region]))
	}
	return amiARNs, nil
}

func (u *Uploader) bucketExists(ctx context.Context) (bool, error) {
	s3C, err := u.s3(ctx)
	if err != nil {
		return false, err
	}
	bucket := u.config.AWS.Bucket
	_, err = s3C.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: &bucket,
	})
	if err == nil {
		return true, nil
	}
	var apiError smithy.APIError
	if errors.As(err, &apiError) && apiError.ErrorCode() == "NotFound" {
		return false, nil
	}
	return false, err
}

func (u *Uploader) ensureBucket(ctx context.Context) error {
	s3C, err := u.s3(ctx)
	if err != nil {
		return err
	}
	bucket := u.config.AWS.Bucket
	exists, err := u.bucketExists(ctx)
	if err != nil {
		return fmt.Errorf("checking if bucket %s exists: %w", bucket, err)
	}
	if exists {
		u.log.Printf("Bucket %s exists", bucket)
		return nil
	}
	u.log.Printf("Bucket %s doesn't exist. Creating.", bucket)
	_, err = s3C.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &bucket,
		CreateBucketConfiguration: &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(u.config.AWS.Region),
		},
	})
	if err != nil {
		return fmt.Errorf("creating bucket %s: %w", bucket, err)
	}
	return nil
}

func (u *Uploader) uploadBlob(ctx context.Context, img io.Reader) error {
	blobName := u.blobName()
	uploadC, err := u.s3uploader(ctx)
	if err != nil {
		return err
	}
	u.log.Printf("Uploading os image as temporary blob %s", blobName)

	_, err = uploadC.Upload(ctx, &s3.PutObjectInput{
		Bucket:            &u.config.AWS.Bucket,
		Key:               &blobName,
		Body:              img,
		ChecksumAlgorithm: s3types.ChecksumAlgorithmSha256,
	})
	return err
}

func (u *Uploader) ensureBlobDeleted(ctx context.Context) error {
	s3C, err := u.s3(ctx)
	if err != nil {
		return err
	}
	bucket := u.config.AWS.Bucket
	blobName := u.blobName()

	bucketExists, err := u.bucketExists(ctx)
	if err != nil {
		return fmt.Errorf("checking if bucket %s exists: %w", bucket, err)
	}
	if !bucketExists {
		u.log.Printf("Bucket %s doesn't exist. Nothing to clean up.", bucket)
		return nil
	}

	_, err = s3C.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &blobName,
	})
	var apiError smithy.APIError
	if errors.As(err, &apiError) && apiError.ErrorCode() == "NotFound" {
		u.log.Printf("Blob %s in %s doesn't exist. Nothing to clean up.", blobName, bucket)
		return nil
	}
	if err != nil {
		return err
	}
	u.log.Printf("Deleting blob %s", blobName)
	_, err = s3C.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &bucket,
		Key:    &blobName,
	})
	return err
}

func (u *Uploader) importSnapshot(ctx context.Context) (string, error) {
	blobName := u.blobName()
	snapshotName := u.snapshotName()
	ec2C, err := u.ec2(ctx, u.config.AWS.Region)
	if err != nil {
		return "", fmt.Errorf("creating ec2 client: %w", err)
	}
	u.log.Printf("Importing %s as snapshot %s", blobName, snapshotName)

	importResp, err := ec2C.ImportSnapshot(ctx, &ec2.ImportSnapshotInput{
		ClientData: &ec2types.ClientData{
			Comment: &snapshotName,
		},
		Description: &snapshotName,
		DiskContainer: &ec2types.SnapshotDiskContainer{
			Description: &snapshotName,
			Format:      toPtr(string(ec2types.DiskImageFormatRaw)),
			UserBucket: &ec2types.UserBucket{
				S3Bucket: &u.config.AWS.Bucket,
				S3Key:    &blobName,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("importing snapshot: %w", err)
	}
	if importResp.ImportTaskId == nil {
		return "", fmt.Errorf("importing snapshot: no import task ID returned")
	}
	u.log.Printf("Waiting for snapshot %s to be ready", snapshotName)
	return waitForSnapshotImport(ctx, ec2C, *importResp.ImportTaskId)
}

func (u *Uploader) ensureSnapshotDeleted(ctx context.Context) error {
	ec2C, err := u.ec2(ctx, u.config.AWS.Region)
	if err != nil {
		return fmt.Errorf("creating ec2 client: %w", err)
	}
	region := u.config.AWS.Region

	snapshots, err := u.findSnapshots(ctx)
	if err != nil {
		return fmt.Errorf("finding snapshots: %w", err)
	}
	for _, snapshot := range snapshots {
		u.log.Printf("Deleting snapshot %s in %s", snapshot, region)
		_, err = ec2C.DeleteSnapshot(ctx, &ec2.DeleteSnapshotInput{
			SnapshotId: toPtr(snapshot),
		})
		if err != nil {
			return fmt.Errorf("deleting snapshot %s: %w", snapshot, err)
		}
	}
	return nil
}

func (u *Uploader) ensureImageDeleted(ctx context.Context, region string) error {
	ec2C, err := u.ec2(ctx, region)
	if err != nil {
		return fmt.Errorf("creating ec2 client: %w", err)
	}
	amiID, err := u.findImage(ctx, region)
	if err == errAMIDoesNotExist {
		u.log.Printf("Image %s in %s doesn't exist. Nothing to clean up.", u.config.Name, region)
		return nil
	}
	snapshotID, err := getBackingSnapshotID(ctx, ec2C, amiID)
	if err == errAMIDoesNotExist {
		u.log.Printf("Image %s doesn't exist. Nothing to clean up.", amiID)
		return nil
	}
	u.log.Printf("Deleting image %s in %s with backing snapshot", amiID, region)
	_, err = ec2C.DeregisterImage(ctx, &ec2.DeregisterImageInput{
		ImageId: &amiID,
	})
	if err != nil {
		return fmt.Errorf("deleting image: %w", err)
	}
	_, err = ec2C.DeleteSnapshot(ctx, &ec2.DeleteSnapshotInput{
		SnapshotId: &snapshotID,
	})
	if err != nil {
		return fmt.Errorf("deleting snapshot: %w", err)
	}
	return nil
}

func (u *Uploader) findSnapshots(ctx context.Context) ([]string, error) {
	ec2C, err := u.ec2(ctx, u.config.AWS.Region)
	if err != nil {
		return nil, fmt.Errorf("creating ec2 client: %w", err)
	}
	snapshots, err := ec2C.DescribeSnapshots(ctx, &ec2.DescribeSnapshotsInput{
		Filters: []ec2types.Filter{
			{
				Name:   toPtr("tag:Name"),
				Values: []string{u.snapshotName()},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("describing snapshots: %w", err)
	}
	var snapshotIDs []string
	for _, s := range snapshots.Snapshots {
		if s.SnapshotId == nil {
			continue
		}
		snapshotIDs = append(snapshotIDs, *s.SnapshotId)
	}
	return snapshotIDs, nil
}

func (u *Uploader) createImageFromSnapshot(ctx context.Context, snapshotID string) (string, error) {
	imageName, err := u.amiName()
	if err != nil {
		return "", fmt.Errorf("inferring image name: %w", err)
	}
	ec2C, err := u.ec2(ctx, u.config.AWS.Region)
	if err != nil {
		return "", fmt.Errorf("creating ec2 client: %w", err)
	}
	u.log.Printf("Creating image %s in %s", imageName, u.config.AWS.Region)

	// TODO(malt3): make UEFI var store configurable (secure boot)
	createReq, err := ec2C.RegisterImage(ctx, &ec2.RegisterImageInput{
		Name:         &imageName,
		Architecture: ec2types.ArchitectureValuesX8664,
		BlockDeviceMappings: []ec2types.BlockDeviceMapping{
			{
				DeviceName: toPtr("/dev/xvda"),
				Ebs: &ec2types.EbsBlockDevice{
					DeleteOnTermination: toPtr(true),
					SnapshotId:          &snapshotID,
				},
			},
		},
		BootMode:           ec2types.BootModeValuesUefi,
		Description:        toPtr(u.config.AWS.AMIDescription),
		EnaSupport:         toPtr(true),
		RootDeviceName:     toPtr("/dev/xvda"),
		TpmSupport:         ec2types.TpmSupportValuesV20,
		VirtualizationType: toPtr("hvm"),
	})
	if err != nil {
		return "", fmt.Errorf("creating image: %w", err)
	}
	if createReq.ImageId == nil {
		return "", fmt.Errorf("creating image: no image ID returned")
	}
	return *createReq.ImageId, nil
}

func (u *Uploader) replicateImage(ctx context.Context, amiID string, targetRegion string) (string, error) {
	imageName, err := u.amiName()
	if err != nil {
		return "", fmt.Errorf("inferring image name: %w", err)
	}
	ec2C, err := u.ec2(ctx, targetRegion)
	if err != nil {
		return "", fmt.Errorf("creating ec2 client: %w", err)
	}
	u.log.Printf("Replicating image %s to %s", imageName, targetRegion)

	replicateReq, err := ec2C.CopyImage(ctx, &ec2.CopyImageInput{
		Name:          &imageName,
		SourceImageId: &amiID,
		SourceRegion:  &u.config.AWS.Region,
	})
	if err != nil {
		return "", fmt.Errorf("replicating image: %w", err)
	}
	if replicateReq.ImageId == nil {
		return "", fmt.Errorf("replicating image: no image ID returned")
	}
	return *replicateReq.ImageId, nil
}

func (u *Uploader) findImage(ctx context.Context, region string) (string, error) {
	ec2C, err := u.ec2(ctx, region)
	if err != nil {
		return "", fmt.Errorf("creating ec2 client: %w", err)
	}
	imageName, err := u.amiName()
	if err != nil {
		return "", fmt.Errorf("inferring image name: %w", err)
	}

	snapshots, err := ec2C.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Filters: []ec2types.Filter{
			{
				Name:   toPtr("name"),
				Values: []string{imageName},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("describing images: %w", err)
	}
	if len(snapshots.Images) == 0 {
		return "", errAMIDoesNotExist
	}
	if len(snapshots.Images) != 1 {
		return "", fmt.Errorf("expected 1 image, got %d", len(snapshots.Images))
	}
	if snapshots.Images[0].ImageId == nil {
		return "", fmt.Errorf("image ID is nil")
	}
	return *snapshots.Images[0].ImageId, nil
}

func (u *Uploader) waitForImage(ctx context.Context, amiID, region string) error {
	u.log.Printf("Waiting for image %s in %s to be created", amiID, region)
	ec2C, err := u.ec2(ctx, region)
	if err != nil {
		return fmt.Errorf("creating ec2 client: %w", err)
	}
	waiter := ec2.NewImageAvailableWaiter(ec2C)
	err = waiter.Wait(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{amiID},
	}, maxWait)
	if err != nil {
		return fmt.Errorf("waiting for image: %w", err)
	}
	return nil
}

func (u *Uploader) tagImageAndSnapshot(ctx context.Context, amiID, region string) error {
	imageName, err := u.amiName()
	if err != nil {
		return fmt.Errorf("inferring image name: %w", err)
	}
	ec2C, err := u.ec2(ctx, region)
	if err != nil {
		return fmt.Errorf("creating ec2 client: %w", err)
	}
	u.log.Printf("Tagging backing snapshot of image %s in %s", amiID, region)
	snapshotID, err := getBackingSnapshotID(ctx, ec2C, amiID)
	if err != nil {
		return fmt.Errorf("getting backing snapshot ID: %w", err)
	}
	_, err = ec2C.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{amiID, snapshotID},
		Tags: []ec2types.Tag{
			{
				Key:   toPtr("Name"),
				Value: toPtr(imageName),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("tagging ami and snapshot: %w", err)
	}
	return nil
}

func (u *Uploader) publishImage(ctx context.Context, amiID, region string) error {
	if !u.config.AWS.Publish {
		return nil
	}

	ec2C, err := u.ec2(ctx, region)
	if err != nil {
		return fmt.Errorf("creating ec2 client: %w", err)
	}
	u.log.Printf("Publishing ami %s in %s", amiID, region)

	_, err = ec2C.ModifyImageAttribute(ctx, &ec2.ModifyImageAttributeInput{
		ImageId: &amiID,
		LaunchPermission: &ec2types.LaunchPermissionModifications{
			Add: []ec2types.LaunchPermission{
				{
					Group: ec2types.PermissionGroupAll,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("publishing image: %w", err)
	}
	return nil
}

func (u *Uploader) accountID(ctx context.Context) (string, error) {
	stsC, err := u.sts(ctx)
	if err != nil {
		return "", fmt.Errorf("creating sts client: %w", err)
	}
	resp, err := stsC.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("getting caller identity: %w", err)
	}
	if resp.Account == nil {
		return "", fmt.Errorf("getting caller identity: no account returned")
	}
	return *resp.Account, nil
}

func (u *Uploader) blobName() string {
	if len(u.config.AWS.BlobName) > 0 {
		return u.config.AWS.BlobName
	}
	return u.config.Name + "-" + u.config.ImageVersion + ".raw"
}

func (u *Uploader) snapshotName() string {
	if len(u.config.AWS.SnapshotName) > 0 {
		return u.config.AWS.SnapshotName
	}
	return u.config.Name + "-" + u.config.ImageVersion
}

func (u *Uploader) amiName() (string, error) {
	type amiNameData struct {
		Name         string
		ImageVersion string
	}
	data := amiNameData{
		Name:         u.config.Name,
		ImageVersion: u.config.ImageVersion,
	}
	amiName := new(strings.Builder)
	if err := u.amiNameTemplate.Execute(amiName, data); err != nil {
		return "", fmt.Errorf("executing ami name template: %w", err)
	}
	return amiName.String(), nil
}

func (u *Uploader) ec2(ctx context.Context, region string) (ec2API, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, err
	}
	return ec2.NewFromConfig(cfg), nil
}

func (u *Uploader) s3(ctx context.Context) (s3API, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(u.config.AWS.Region))
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(cfg), nil
}

func (u *Uploader) s3uploader(ctx context.Context) (s3UploaderAPI, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(u.config.AWS.Region))
	if err != nil {
		return nil, err
	}
	return s3manager.NewUploader(s3.NewFromConfig(cfg)), nil
}

func (u *Uploader) sts(ctx context.Context) (stsAPI, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(u.config.AWS.Region))
	if err != nil {
		return nil, err
	}
	return sts.NewFromConfig(cfg), nil
}

func waitForSnapshotImport(ctx context.Context, ec2C ec2API, importTaskID string) (string, error) {
	start := time.Now()
	for {
		if time.Since(start) > maxWait {
			return "", fmt.Errorf("importing snapshot: timeout")
		}
		taskResp, err := ec2C.DescribeImportSnapshotTasks(ctx, &ec2.DescribeImportSnapshotTasksInput{
			ImportTaskIds: []string{importTaskID},
		})
		if err != nil {
			return "", fmt.Errorf("describing import snapshot task: %w", err)
		}
		if len(taskResp.ImportSnapshotTasks) == 0 {
			return "", fmt.Errorf("describing import snapshot task: no tasks returned")
		}
		if taskResp.ImportSnapshotTasks[0].SnapshotTaskDetail == nil {
			return "", fmt.Errorf("describing import snapshot task: no snapshot task detail returned")
		}
		if taskResp.ImportSnapshotTasks[0].SnapshotTaskDetail.Status == nil {
			return "", fmt.Errorf("describing import snapshot task: no status returned")
		}
		var statusMessage string
		if taskResp.ImportSnapshotTasks[0].SnapshotTaskDetail.StatusMessage != nil {
			statusMessage = *taskResp.ImportSnapshotTasks[0].SnapshotTaskDetail.StatusMessage
		}
		switch *taskResp.ImportSnapshotTasks[0].SnapshotTaskDetail.Status {
		case string(ec2types.SnapshotStatePending):
			// continue waiting
		case string("active"):
			// continue waiting
		case string(ec2types.SnapshotStateCompleted):
			// done
			return *taskResp.ImportSnapshotTasks[0].SnapshotTaskDetail.SnapshotId, nil
		case string(ec2types.SnapshotStateError):
			return "", fmt.Errorf("importing snapshot: task failed with message %q", statusMessage)
		case string("deleted"):
			log.Printf("Importing snapshot failed with \"deleted\" status. This may indicate a missing service role for the AWS service \"vmie.amazonaws.com\" to access the snapshot. See https://docs.aws.amazon.com/vm-import/latest/userguide/required-permissions.html#vmimport-role for details.")
			return "", fmt.Errorf("importing snapshot: import state deleted with message %q", statusMessage)
		default:
			return "", fmt.Errorf("importing snapshot: status %s with message %q",
				*taskResp.ImportSnapshotTasks[0].SnapshotTaskDetail.Status,
				statusMessage,
			)
		}
		time.Sleep(waitInterval)
	}
}

func getBackingSnapshotID(ctx context.Context, ec2C ec2API, amiID string) (string, error) {
	describeResp, err := ec2C.DescribeImages(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{amiID},
	})
	if err != nil || len(describeResp.Images) == 0 {
		return "", errAMIDoesNotExist
	}
	if len(describeResp.Images) != 1 {
		return "", fmt.Errorf("describing image: expected 1 image, got %d", len(describeResp.Images))
	}
	image := describeResp.Images[0]
	if len(image.BlockDeviceMappings) != 1 {
		return "", fmt.Errorf("found %d block device mappings for image %s, expected 1", len(image.BlockDeviceMappings), amiID)
	}
	if image.BlockDeviceMappings[0].Ebs == nil {
		return "", fmt.Errorf("image %s does not have an EBS block device mapping", amiID)
	}
	ebs := image.BlockDeviceMappings[0].Ebs
	if ebs.SnapshotId == nil {
		return "", fmt.Errorf("image %s does not have an EBS snapshot", amiID)
	}
	return *ebs.SnapshotId, nil
}

// getAMIARN returns the arn of the AMI with the given region, account ID and ami ID.
func getAMIARN(region, accountID, amiID string) string {
	return fmt.Sprintf("arn:aws:ec2:%s:%s:image/%s", region, accountID, amiID)
}

func toPtr[T any](v T) *T {
	return &v
}
