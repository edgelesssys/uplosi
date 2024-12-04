package config

import future.keywords.in

deny[msg] {
    not input.Provider in valid_csps

    msg = sprintf("cloud provider %q unknown", [input.Provider])
}

deny[msg] {
    not regex.match(`^\d+\.\d+\.\d+$`, input.ImageVersion)

    msg = sprintf("image version %q must be in format <MAJOR>.<MINOR>.<PATCH>", [input.ImageVersion])
}

deny[msg] {
    input.Name == ""

    msg = "required field name empty"
}

deny[msg] {
    input.Provider == "aws"
    some "" in input.AWS.ReplicationRegions

    msg = "member of list replicationRegions empty for provider aws"
}

deny[msg] {
    input.Provider == "aws"
    input.AWS.AMIName != ""
    not length_in_range(input.AWS.AMIName, 3, 128)

    msg = sprintf("field amiName must be between 3 and 128 characters for provider aws, got %d", [count(input.AWS.AMIName)])
}

deny[msg] {
    input.Provider == "aws"
    input.AWS.AMIName != ""
    not regex.match(`^[a-zA-Z0-9().\-/_]+$`, input.AWS.AMIName)

    msg = sprintf("ami name %q should only contain letters, numbers, '(', ')', '.', '-', '/' and '_'", [input.AWS.AMIName])
}

# https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html - 1
deny[msg] {
    input.Provider == "aws"
    input.AWS.Bucket != ""
    not length_in_range(input.AWS.Bucket, 3, 63)

    msg = sprintf("field bucket must be between 3 and 63 characters for provider aws, got %d", [count(input.AWS.Bucket)])
}

# https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html - 2
deny[msg] {
    input.Provider == "aws"
    input.AWS.Bucket != ""
    not regex.match(`^[a-z0-9.\-]+$`, input.AWS.Bucket)

    msg = sprintf("bucket name %q should only contain lowercase letters, numbers, dots (.) and hyphens", [input.AWS.Bucket])
}

# https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html - 3
deny[msg] {
    input.Provider == "aws"
    input.AWS.Bucket != ""
    not begin_and_end_with(input.AWS.Bucket, lowercase_letters | digits)

    msg = sprintf("bucket name %q must begin and end with a letter or number", [input.AWS.Bucket])
}

# https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html - 4
deny[msg] {
    input.Provider == "aws"
    regex.match(`[.]{2}`, input.AWS.Bucket)

    msg = sprintf("bucket name %q must not contain two adjacent periods", [input.AWS.Bucket])
}

# https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html - 6
deny[msg] {
    input.Provider == "aws"
    regex.match(`^xn--`, input.AWS.Bucket)

    msg = sprintf("bucket name %q must not start with the prefix xn--", [input.AWS.Bucket])
}

# https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html - 7
deny[msg] {
    input.Provider == "aws"
    regex.match(`^sthree-`, input.AWS.Bucket)

    msg = sprintf("bucket name %q must not start with the prefix sthree- and the prefix sthree-configurator", [input.AWS.Bucket])
}

# https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html - 8
deny[msg] {
    input.Provider == "aws"
    regex.match(`-s3alias$`, input.AWS.Bucket)

    msg = sprintf("bucket name %q must not end with the suffix -s3alias", [input.AWS.Bucket])
}

# https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html - 9
deny[msg] {
    input.Provider == "aws"
    regex.match(`--ol-s3$`, input.AWS.Bucket)

    msg = sprintf("bucket name %q must not end with the suffix --ol-s3", [input.AWS.Bucket])
}

deny[msg] {
    input.Provider == "aws"
    input.AWS.BucketLocationConstraint in [
		"af-south-1",
		"ap-east-1",
		"ap-northeast-1",
		"ap-northeast-2",
		"ap-northeast-3",
		"ap-south-1",
		"ap-southeast-1",
		"ap-southeast-2",
		"ap-southeast-3",
		"ca-central-1",
		"cn-north-1",
		"cn-northwest-1",
		"EU",
		"eu-central-1",
		"eu-north-1",
		"eu-south-1",
		"eu-west-1",
		"eu-west-2",
		"eu-west-3",
		"me-south-1",
		"sa-east-1",
		"us-east-2",
		"us-gov-east-1",
		"us-gov-west-1",
		"us-west-1",
		"us-west-2",
		"ap-south-2",
		"eu-south-2",
    ]

    msg = sprintf("%q is not a valid bucket location constraint", [ input.AWS.BucketLocationConstraint ] )
}

deny[msg] {
    input.Provider == "aws"
    not is_boolean(input.AWS.Publish)

    msg = "required field Publish uninitialized for provider aws"
}

deny[msg] {
    input.Provider == "azure"
    input.Azure.SubscriptionID != ""
    not regex.match(`^(?:\{{0,1}(?:[0-9a-fA-F]){8}-(?:[0-9a-fA-F]){4}-(?:[0-9a-fA-F]){4}-(?:[0-9a-fA-F]){4}-(?:[0-9a-fA-F]){12}\}{0,1})$$`, input.Azure.SubscriptionID)

    msg = sprintf("subscription id %q must be a valid guid for provider azure", [input.Azure.SubscriptionID])
}

deny[msg] {
    input.Provider == "azure"
    input.Azure.AttestationVariant != ""
    not input.Azure.AttestationVariant in ["azure-tdx", "azure-sev-snp", "azure-trustedlaunch"]

    msg = sprintf("attestation variant %q must be one of %s for provider azure", [input.Azure.AttestationVariant, ["azure-tdx", "azure-sev-snp", "azure-trustedlaunch"]])
}

deny[msg] {
    input.Provider == "azure"
    input.Azure.SharedImageGallery != ""
    not regex.match(`^[a-zA-Z0-9_.]*$`, input.Azure.SharedImageGallery)

    msg = sprintf("shared image gallery %q must contain only alphanumerics, underscores and periods for provider azure", [input.Azure.SharedImageGallery])
}

deny[msg] {
    input.Provider == "azure"
    input.Azure.SharedImageGallery != ""
    not begin_and_end_with(input.Azure.SharedImageGallery, lowercase_letters | uppercase_letters | digits)

    msg = sprintf("shared image gallery %q must begin and end with a letter or number", [input.Azure.SharedImageGallery])
}

deny[msg] {
    input.Provider == "azure"
    input.Azure.SharedImageGallery != ""
    not length_in_range(input.Azure.SharedImageGallery, 1, 80)

    msg = sprintf("field sharedImageGallery must be between 1 and 80 characters for provider azure, got %d", [count(input.Azure.SharedImageGallery)])
}

deny[msg] {
    input.Provider == "azure"
    input.Azure.SharingProfile != ""
    allowed := ["community", "private"]
    not input.Azure.SharingProfile in allowed

    msg = sprintf("sharing profile %q must be one of %s for provider azure", [input.Azure.SharingProfile, allowed])
}

deny[msg] {
    input.Provider == "azure"
    input.Azure.SharingProfile == "community"
    input.Azure.SharingNamePrefix == ""

    msg = "field sharingNamePrefix is required for sharing profile community and provider azure"
}

deny[msg] {
    input.Provider == "azure"
    input.Azure.SharingNamePrefix != ""
    not length_in_range(input.Azure.SharingNamePrefix, 5, 16)

    msg = sprintf("field sharingNamePrefix must be between 5 and 16 characters for provider azure, got %d", [count(input.Azure.SharingNamePrefix)])
}

deny[msg] {
    input.Provider == "azure"
    input.Azure.SharingNamePrefix != ""
    not regex.match(`^[a-zA-Z0-9]*$`, input.Azure.SharingNamePrefix)

    msg = sprintf("sharing name prefix %q must be alphanumeric for provider azure", [input.Azure.SharingNamePrefix])
}

deny[msg] {
    input.Provider == "azure"
    input.Azure.ImageDefinitionName != ""
    not regex.match(`^[a-zA-Z0-9_\-.]*$`, input.Azure.ImageDefinitionName)

    msg = sprintf("image definition name %q must contain only alphanumerics, underscores, hyphens, and periods for provider azure", [input.Azure.ImageDefinitionName])
}

deny[msg] {
    input.Provider == "azure"
    input.Azure.ImageDefinitionName != ""
    not begin_and_end_with(input.Azure.ImageDefinitionName, lowercase_letters | uppercase_letters | digits)

    msg = sprintf("image definition name %q must begin and end with a letter or number", [input.Azure.ImageDefinitionName])
}

deny[msg] {
    input.Provider == "azure"
    input.Azure.ImageDefinitionName != ""
    not length_in_range(input.Azure.ImageDefinitionName, 1, 80)

    msg = sprintf("field imageDefinitionName must be between 1 and 80 characters for provider azure, got %d", [count(input.Azure.ImageDefinitionName)])
}

deny[msg] {
    input.Provider == "azure"
    input.Azure.DiskName != ""
    not regex.match(`^[a-zA-Z0-9_\-.]*$`, input.Azure.DiskName)

    msg = sprintf("disk name %q must contain only alphanumerics, underscores, hyphens, and periods for provider azure", [input.Azure.DiskName])
}

deny[msg] {
    input.Provider == "azure"
    input.Azure.DiskName != ""
    not length_in_range(input.Azure.DiskName, 1, 80)

    msg = sprintf("field diskName must be between 1 and 80 characters for provider azure, got %d", [count(input.Azure.DiskName)])
}

deny[msg] {
    input.Provider == "gcp"
    input.GCP.Project != ""
    not regex.match(`^[a-z0-9\-]*$`, input.GCP.Project)

    msg = sprintf("project name %q must contain only lowercase letters, digits and hyphens for provider gcp", [input.GCP.Project])
}

deny[msg] {
    input.Provider == "gcp"
    input.GCP.Project != ""
    not begins_with(input.GCP.Project, lowercase_letters)
    not ends_with(input.GCP.Project, lowercase_letters | digits)

    msg = sprintf("project name %q must begin with a letter and end with a letter or number", [input.GCP.Project])
}

deny[msg] {
    input.Provider == "gcp"
    input.GCP.Project != ""
    not length_in_range(input.GCP.Project, 6, 30)

    msg = sprintf("field project must be between 6 and 30 characters for provider gcp, got %d", [count(input.GCP.Project)])
}

deny[msg] {
    input.Provider == "gcp"
    input.GCP.ImageName != ""
    not regex.match(`^[a-z0-9\-]*$`, input.GCP.ImageName)

    msg = sprintf("image name %q must contain only alphanumerics, underscores, hyphens, and periods for provider gcp", [input.GCP.ImageName])
}

deny[msg] {
    input.Provider == "gcp"
    input.GCP.ImageName != ""
    not begins_with(input.GCP.ImageName, lowercase_letters)
    not ends_with(input.GCP.ImageName, lowercase_letters | digits)

    msg = sprintf("image name %q must begin with a letter and end with a letter or number", [input.GCP.ImageName])
}

deny[msg] {
    input.Provider == "gcp"
    input.GCP.ImageName != ""
    not length_in_range(input.GCP.ImageName, 1, 63)

    msg = sprintf("field imageName must be between 1 and 63 characters for provider gcp, got %d", [count(input.GCP.ImageName)])
}

deny[msg] {
    input.Provider == "gcp"
    input.GCP.ImageFamily != ""
    not regex.match(`^[a-z0-9\-]*$`, input.GCP.ImageFamily)

    msg = sprintf("image family %q must contain only alphanumerics, underscores, hyphens, and periods for provider gcp", [input.GCP.ImageFamily])
}

deny[msg] {
    input.Provider == "gcp"
    input.GCP.ImageFamily != ""
    not begins_with(input.GCP.ImageFamily, lowercase_letters)
    not ends_with(input.GCP.ImageFamily, lowercase_letters | digits)

    msg = sprintf("image family %q must begin with a letter and end with a letter or number", [input.GCP.ImageFamily])
}

deny[msg] {
    input.Provider == "gcp"
    input.GCP.ImageFamily != ""
    not length_in_range(input.GCP.ImageFamily, 1, 63)

    msg = sprintf("field imageFamily must be between 1 and 63 characters for provider gcp, got %d", [count(input.GCP.ImageFamily)])
}

deny[msg] {
    input.Provider == "gcp"
    input.GCP.Bucket != ""
    not regex.match(`^[a-z0-9\-_.]*$`, input.GCP.Bucket)

    msg = sprintf("bucket %q must contain only alphanumerics, underscores, hyphens, and periods for provider gcp", [input.GCP.Bucket])
}

deny[msg] {
    input.Provider == "gcp"
    input.GCP.Bucket != ""
    not begin_and_end_with(input.GCP.Bucket, lowercase_letters | digits)

    msg = sprintf("bucket %q must begin with a letter and end with a letter or number", [input.GCP.Bucket])
}

deny[msg] {
    input.Provider == "gcp"
    input.GCP.Bucket != ""
    not length_in_range(input.GCP.Bucket, 3, 63)

    msg = sprintf("field bucket must be between 1 and 63 characters for provider gcp, got %d", [count(input.GCP.Bucket)])
}

deny[msg] {
    input.Provider == "openstack"
    input.OpenStack.Visibility != ""
    allowed := ["public", "private", "shared", "community"]
    not input.OpenStack.Visibility in allowed

    msg = sprintf("field visibility must be one of %s for provider openstack", allowed)
}

deny[msg] {
    some provider in valid_csps
    input.Provider == provider
    some fieldName, fieldValue in required_fields[provider]
    fieldValue == ""

    msg = sprintf("required field %q empty for provider %s", [fieldName, input.Provider])
}

length_in_range(s, min_len, max_len) = in_range {
    length := count(s)
    in_range := all([min_len <= length, length <= max_len])
}

begins_with(s, charset) = begin {
    begin := substring(s, 0, 1) in charset
}

ends_with(s, charset) = end {
    end :=  substring(s, count(s)-1, 1) in charset
}

begin_and_end_with(s, charset) = begin_and_end {
    begin_and_end := all([
        begins_with(s, charset),
        ends_with(s, charset),
    ])
}

valid_csps := [ "aws", "azure", "gcp", "openstack" ]

required_fields := {
    "aws": {
        "region": input.AWS.Region,
        "replicationRegions": input.AWS.ReplicationRegions,
        "amiName": input.AWS.AMIName,
        "bucket": input.AWS.Bucket,
        "blobName": input.AWS.BlobName,
        "snapshotName": input.AWS.SnapshotName,
    },
    "azure": {
        "subscriptionID": input.Azure.SubscriptionID,
        "location": input.Azure.Location,
        "replicationRegions": input.Azure.ReplicationRegions,
        "resourceGroup": input.Azure.ResourceGroup,
        "attestationVariant": input.Azure.AttestationVariant,
        "sharedImageGallery": input.Azure.SharedImageGallery,
        "sharingProfile": input.Azure.SharingProfile,
        "imageDefinitionName": input.Azure.ImageDefinitionName,
        "diskName": input.Azure.DiskName,
        "offer": input.Azure.Offer,
        "sku": input.Azure.SKU,
        "publisher": input.Azure.Publisher,
        "diskName": input.Azure.DiskName,
    },
    "gcp": {
        "project": input.GCP.Project,
        "location": input.GCP.Location,
        "imageName": input.GCP.ImageName,
        "imageFamily": input.GCP.ImageFamily,
        "bucket": input.GCP.Bucket,
        "blobName": input.GCP.BlobName,
    },
    "openstack": {
        "cloud": input.OpenStack.Cloud,
        "imageName": input.OpenStack.ImageName,
    },
}

lowercase_letters := {
    "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"
}

uppercase_letters := {
    "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"
}

digits := { "0", "1", "2", "3", "4", "5", "6", "7", "8", "9"  }
