/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/

package azure

import "strings"

//go:generate stringer -type=DiskType -trimprefix=DiskType

// DiskType is the kind of disk created using the Azure API.
type DiskType uint32

// FromString converts a string into an DiskType.
func FromString(s string) DiskType {
	switch strings.ToLower(s) {
	case strings.ToLower(DiskTypeNormal.String()):
		return DiskTypeNormal
	case strings.ToLower(DiskTypeWithVMGS.String()):
		return DiskTypeWithVMGS
	default:
		return DiskTypeUnknown
	}
}

const (
	// DiskTypeUnknown is default value for DiskType.
	DiskTypeUnknown DiskType = iota
	// DiskTypeNormal creates a normal Azure disk (single block device).
	DiskTypeNormal
	// DiskTypeWithVMGS creates a disk with VMGS (also called secure disk)
	// that has an additional block device for the VMGS disk.
	DiskTypeWithVMGS
)
