package testdata

import _ "embed"

// UKI returns the UKI EFI binary.
func UKI() []byte {
	return ukiEFI[:]
}

//go:embed uki.efi
var ukiEFI []byte
