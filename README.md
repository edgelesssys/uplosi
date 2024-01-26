# Uplosi — Upload OS images to cloud provider


# Installation

With Go installed on your system, run the following command:

```shell-session
go install github.com/edgelesssys/uplosi@latest
```

Alternatively, you can download the binary from the [releases page](https://github.com/edgelesssys/uplosi/releases/latest).

# Usage

```shell-session
uplosi <image> [flags]
```

## Examples

```shell-session
# edit uplosi.conf, then run
uplosi image.raw -i
```

## Flags

- `--disable-variant-glob` string: list of variant name globs to disable
- `--enable-variant-glob` string: list of variant name globs to enable
- `-h`,`--help`: help for uplosi
- `-i`,`--increment-version`: increment version number after upload
- `-v`: version for uplosi

# Configuration

Uplosi requires configuration files in [TOML format](https://toml.io/en/) to be present in the user's workspace (CWD).
Namely, the following files are read:

- `uplosi.conf` — the main configuration file, read first
- `uplosi.conf.d/*.conf` — (optional) additional configuration files, read in alphabetical order


Any settings specified in the additional configuration files will override the settings specified in the main configuration file.
The configuration has the following structure:

```toml
[base]

# Base configuration that is applied to every variant.

[variant.<name>] # e.g. variant.default

# Variant specific configuration that overrides the base configuration.
```

## Example

```toml
[base]
imageVersion = "1.2.3"
name = "my-image"

[base.aws]
# AWS specific configuration that is applied to every variant.
region = "eu-central-1"
replicationRegions = ["us-east-2", "ap-south-1"]
bucket = "my-bucket"
publish = true

[base.azure]
# Azure specific configuration that is applied to every variant.
subscriptionID = "00000000-0000-0000-0000-000000000000"
location = "northeurope"
resourceGroup = "my-rg"
sharedImageGallery = "my_gallery"
sharingNamePrefix = "my-shared-"

[base.gcp]
# GCP specific configuration that is applied to every variant.
project = "myproject-123456"
location = "europe-west3"
bucket = "my-bucket"

[variant.foo]
# Variant specific configuration that overrides the base configuration.
provider = "aws"
imageVersionFile = "foo-version.txt" # overrides base.imageVersion

[variant.foo.aws]
# Variant specific configuration that overrides the base.aws configuration.
replicationRegions = []
publish = false

[variant.bar]
# Variant specific configuration that overrides the base configuration.
provider = "azure"

[variant.bar.azure]
# Variant specific configuration that overrides the base.azure configuration.
resourceGroup = "my-rg-bar" # overrides base.azure.resourceGroup
```

## Reference

The following settings are supported:

### `base.provider` / `variant.<name>.provider`

- Default: none
- Required: yes

The cloud provider to upload the image to: `aws`, `azure` or `gcp`.

### `base.imageVersion` / `variant.<name>.imageVersion`

- Default: `"0.0.0"`
- Required: no

A version string with the format `<major>.<minor>.<patch>`, e.g. `1.0.0`.
This version string can be used as a template parameter `{{.Version}}` in all template strings.
Additionally, the individual version components can be accessed via `{{.VersionMajor}}`, `{{.VersionMinor}}` and `{{.VersionPatch}}`.

### `base.imageVersionFile` / `variant.<name>.imageVersionFile`

- Default: none
- Required: no

A file to read the image version from. The file must contain a single line with the image version string.
If set, the file contents will overwrite the `imageVersion` setting.
When using the `-i` / `--increment-version` command line option, the version will be incremented after uploading and written back to the file.

### `base.name` / `variant.<name>.name`

- Default: none
- Required: yes

The name of the image to upload. This name can be used as a template parameter `{{.Name}}` in all template strings.

### `base.aws.region` / `variant.<name>.aws.region`

- Default: none
- Required: yes

The primary AWS region to upload the ami to. Example: `eu-central-1`.
This region is used for the S3 bucket, EBS snapshot and the primary AMI.
Subsequent AMIs are copied to all other regions specified in `replicationRegions`.

### `base.aws.replicationRegions` / `variant.<name>.aws.replicationRegions`

- Default: `[]`
- Required: no

Additional AWS regions that the ami will be replicated in. Example: `["us-east-2", "ap-south-1"]`.

### `base.aws.amiName` / `variant.<name>.aws.amiName`

- Default: `"{{.Name}}-{{.Version}}"`
- Required: no
- Template: yes

The name of the AMI.

### `base.aws.amiDescription` / `variant.<name>.aws.amiDescription`

- Default: `"{{.Name}}-{{.Version}}"`
- Required: no
- Template: yes

The description of the AMI.

### `base.aws.bucket` / `variant.<name>.aws.bucket`

- Default: none
- Required: yes
- Template: yes

The bucket to upload the image to during the upload process.

### `base.aws.bucketRegionConstraint` / `variant.<name>.aws.bucketRegionConstraint`

- Default: none (defaults to `us-east-1`)
- Required: no
- Template: no

The region where the buckets exist or should be created.

### `base.aws.blobName` / `variant.<name>.aws.blobName`

- Default: `"{{.Name}}-{{.Version}}.raw"`
- Required: no
- Template: yes

Name of temporary blob within `bucket`. Image is uploaded to this blob before being converted to an AMI.

### `base.aws.snapshotName` / `variant.<name>.aws.snapshotName`

- Default: `"{{.Name}}-{{.Version}}"`
- Required: no
- Template: yes

Name of the EBS snapshot that is the backing store for the AMI.

### `base.aws.publish` / `variant.<name>.aws.publish`

- Default: `false`
- Required: no

If set, the AMI will be published (made publicly available) after uploading.

### `base.azure.subscriptionID` / `variant.<name>.azure.subscriptionID`

- Default: none
- Required: yes

Id of the Azure subscription to upload the image to. Use `az account subscription list` to list all available subscriptions.

### `base.azure.location` / `variant.<name>.azure.location`

- Default: none
- Required: yes

The primary Azure region to upload the image to. Example: `northeurope`.
This region is used for the resource group, disk and gallery.
Subsequent images are replicated to all other regions specified in `replicationRegions`.

### `base.azure.replicationRegions` / `variant.<name>.azure.replicationRegions`

- Default: `[]`
- Required: no

Additional Azure regions that the image will be replicated in. Example: `["northeurope", "eastus2"]`.

### `base.azure.resourceGroup` / `variant.<name>.azure.resourceGroup`

- Default: none
- Required: yes
- Template: yes

The resource group to create the image and any additional resources in.
Will be created if it does not exist. Example: `"my-rg"`.

### `base.azure.attestationVariant` / `variant.<name>.azure.attestationVariant`

- Default: `"azure-sev-snp"`
- Required: no
- Template: yes

The attestation variant to use. One of `azure-tdx`, `azure-sev-snp`, `azure-trustedlaunch`.
Used to determine the security type of the image.

### `base.azure.sharedImageGallery` / `variant.<name>.azure.sharedImageGallery`

- Default: none
- Required: yes
- Template: yes

Name of the shared image gallery to upload the image to. Example: `"my_gallery"`.
Will be created if it does not exist.

### `base.azure.sharingProfile` / `variant.<name>.azure.sharingProfile`

- Default: `"community"`
- Required: no
- Template: yes

Sharing profile to use for the gallery image. One of `community`, `groups`, `private`.
Community images are publicly available, group images are available to a specific group of users, private images are only available to the owner.

### `base.azure.sharingNamePrefix` / `variant.<name>.azure.sharingNamePrefix`

- Default: none
- Required: if `sharingProfile` is `community` or `groups`
- Template: yes

Prefix for the shared image name. Example: `"my-image"`.
The full name will contain the prefix with a random suffix.

### `base.azure.imageDefinitionName` / `variant.<name>.azure.imageDefinitionName`

- Default: `"{{.Name}}"`
- Required: no
- Template: yes

Name of the image definition within the shared image gallery.

### `base.azure.offer` / `variant.<name>.azure.offer`

- Default: `"Linux"`
- Required: no
- Template: yes

The name of a group of related images created by a publisher. Examples: `"UbuntuServer"`, `"FedoraServer"`.

### `base.azure.sku` / `variant.<name>.azure.sku`

- Default: `"{{.Name}}-{{.VersionMajor}}"`
- Required: no
- Template: yes

An instance of an offer, such as a major release of a distribution. Example: `"18.04-LTS"`.

### `base.azure.publisher` / `variant.<name>.azure.publisher`

- Default: `"Contoso"`
- Required: no
- Template: yes

The organization that created the image. Example: `"Edgeless Systems"`.

### `base.azure.diskName` / `variant.<name>.azure.diskName`

- Default: `"{{.Name}}-{{.Version}}"`
- Required: no
- Template: yes

Name of the temporary disk. Image is uploaded to this disk before being converted to an image.

### `base.gcp.project` / `variant.<name>.gcp.project`

- Default: none
- Required: yes

Name of the GCP project to upload the image to. Example: `"my-project"`.
Can be retrieved with `gcloud config get-value project`.

### `base.gcp.location` / `variant.<name>.gcp.location`

- Default: none
- Required: yes

Location of the GCP project to create resources in. Example: `"europe-west3"`.
Images will be accessible globally, regardless of the location setting.

### `base.gcp.imageName` / `variant.<name>.gcp.imageName`

- Default: `"{{.Name}}-{{replaceAll .Version \".\" \"-\"}}"`
- Required: no
- Template: yes

Name of the image to create. Example: `"my-image-1-0-0"`.

### `base.gcp.imageFamily` / `variant.<name>.gcp.imageFamily`

- Default: `"{{.Name}}"`
- Required: no
- Template: yes

Family that the image belongs to. Example: `"my-image"`.

### `base.gcp.bucket` / `variant.<name>.gcp.bucket`

- Default: none
- Required: yes
- Template: yes

Name of the GCS bucket to upload the image to temporarily. Example: `"my-bucket"`.
Will be created if it does not exist.

### `base.gcp.blobName` / `variant.<name>.gcp.blobName`

- Default: `"{{.Name}}-{{.Version}}.tar.gz"`
- Required: no
- Template: yes

Name of the temporary blob within `bucket`. Image is uploaded to this blob before being converted to an image.
