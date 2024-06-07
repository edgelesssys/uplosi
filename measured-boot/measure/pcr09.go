/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: AGPL-3.0-only
*/

package measure

import (
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/text/encoding/unicode"
)

// DescribeLinuxLoad2 describes the expected measurements for the Linux LOAD_FILE2 protocol.
func DescribeLinuxLoad2(w io.Writer, cmdline []byte, initrdDigest [32]byte) error {
	if _, err := fmt.Fprintf(w, "Linux LOAD_FILE2 protocol:\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  cmdline: %q\n", cmdline); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  initrd (digest %x)\n", initrdDigest); err != nil {
		return err
	}
	return nil
}

// PredictPCR9 predicts the PCR9 value based on the kernel command line and initrd.
func PredictPCR9(simulator *Simulator, cmdline []byte, initrdDigest [32]byte) error {
	// Linux LOAD_FILE2 protocol

	// Some UKI builders will not null-terminate the command line, so we do it here.
	// See: https://github.com/systemd/mkosi/blob/abef37482330e5b3fdc8ba72bff0bdcedbf6006d/mkosi/__init__.py#L2030
	if len(cmdline) == 0 || cmdline[len(cmdline)-1] != 0 {
		cmdline = append(cmdline, 0)
	}

	cmdlineUTF16LE, err := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder().Bytes(cmdline)
	if err != nil {
		return err
	}
	err = simulator.ExtendPCR(9, sha256.Sum256(cmdlineUTF16LE), cmdlineUTF16LE, fmt.Sprintf("EV_EVENT_TAG: Linux LOAD_FILE2 protocol: cmdline %q", cmdline))
	if err != nil {
		return err
	}

	// Linux LOAD_FILE2 protocol - efi_load_initrd
	// https://github.com/torvalds/linux/blob/42dc814987c1feb6410904e58cfd4c36c4146150/drivers/firmware/efi/libstub/efi-stub-helper.c#L559
	// initrd is hashed as-is and measured
	err = simulator.ExtendPCR(9, initrdDigest, nil, fmt.Sprintf("EV_EVENT_TAG: Linux LOAD_FILE2 protocol: initrd (digest %x)", initrdDigest))
	if err != nil {
		return err
	}

	return nil
}
