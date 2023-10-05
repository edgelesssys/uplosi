/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/
package azure

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"
)

const (
	sectorSize         = 512
	vhdFixedHeaderSize = 512
	dataAlignmentBytes = 1048576 // 1 MiB
)

// vhdReader is a reader for raw image files
// it appends the VHD footer to the end of the image
// and pads the image to a multiple of 1 MiB
type vhdReader struct {
	data        io.Reader
	payloadSize uint64
	footer      [vhdFixedHeaderSize]byte

	pos uint64
}

func newVHDReader(data io.Reader, size uint64, uuid [16]byte, timestamp time.Time) *vhdReader {
	footer := newVHDFixedHeader(size, uuid, timestamp)

	reader := &vhdReader{
		data:        data,
		payloadSize: size,
	}
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, footer); err != nil {
		panic(err)
	}
	copy(reader.footer[:], buf.Bytes())

	return reader
}

// PayloadSize returns the size of the payload (i.e. the raw image data)
func (r *vhdReader) PayloadSize() uint64 {
	return r.payloadSize
}

// ContainerSize returns the size of the container (i.e. the raw image data plus
// padding and the VHD footer)
func (r *vhdReader) ContainerSize() uint64 {
	padSize := dataAlignmentBytes - (r.payloadSize % dataAlignmentBytes)
	return r.payloadSize + padSize + vhdFixedHeaderSize
}

func (r *vhdReader) Read(p []byte) (int, error) {
	// first, read the data unmodified
	if r.pos < r.payloadSize {
		read, err := r.data.Read(p)
		if err == io.EOF {
			r.pos = r.payloadSize
			return read, nil
		}
		if err != nil {
			return read, err
		}
		r.pos += uint64(read)
		return read, nil
	}

	// then, pad the data to a multiple of 1 MiB
	padSize := dataAlignmentBytes - (r.payloadSize % dataAlignmentBytes)
	padEnd := r.payloadSize + padSize
	padNow := min(padEnd-r.pos, uint64(len(p)))
	if r.pos < padEnd {
		for i := uint64(0); i < padNow; i++ {
			p[i] = 0
		}
		r.pos += padNow
		return int(padNow), nil
	}

	// finally, append the VHD footer
	footerPos := r.pos - padEnd
	footerRange := min(len(p), vhdFixedHeaderSize)
	if r.pos < padEnd+vhdFixedHeaderSize {
		copy(p, r.footer[footerPos:footerRange])
		r.pos += uint64(footerRange)
		return footerRange, nil
	}

	return 0, io.EOF
}

// VHDFixedHeader is the fixed size (512 byte) trailing header of a VHD file.
type VHDFixedHeader struct {
	Cookie             [8]byte   // offset 0
	Features           [4]byte   // offset 8
	FileFormatVersion  [4]byte   // offset 12
	DataOffset         [8]byte   // offset 16
	Timestamp          [4]byte   // offset 24
	CreatorApplication [4]byte   // offset 28
	CreatorVersion     [4]byte   // offset 32
	CreatorHostOS      [4]byte   // offset 36
	OriginalSize       [8]byte   // offset 40
	CurrentSize        [8]byte   // offset 48
	DiskGeometry       [4]byte   // offset 56
	DiskType           [4]byte   // offset 60
	Checksum           [4]byte   // offset 64
	UniqueId           [16]byte  // offset 68
	SavedState         [1]byte   // offset 84
	Reserved           [427]byte // offset 85
}

func newVHDFixedHeader(size uint64, uuid [16]byte, timestamp time.Time) VHDFixedHeader {
	sizeWithPadding := size + (dataAlignmentBytes - (size % dataAlignmentBytes))

	var header VHDFixedHeader
	copy(header.Cookie[:], "conectix")
	copy(header.Features[:], "\x00\x00\x00\x02")
	copy(header.FileFormatVersion[:], "\x00\x01\x00\x00")
	copy(header.DataOffset[:], "\xff\xff\xff\xff\xff\xff\xff\xff")

	formattedTimestamp := uint32(timestamp.Unix()) - 946684800
	binary.BigEndian.PutUint32(header.Timestamp[:], formattedTimestamp)
	copy(header.CreatorApplication[:], "uplo")
	copy(header.CreatorHostOS[:], "Win2k")
	binary.BigEndian.PutUint64(header.OriginalSize[:], sizeWithPadding)
	binary.BigEndian.PutUint64(header.CurrentSize[:], sizeWithPadding)

	totalSectors := sizeWithPadding / sectorSize
	cylinders, heads, sectorsPerTrack := calculateCHS(totalSectors)
	binary.BigEndian.PutUint16(header.DiskGeometry[:2], cylinders)
	header.DiskGeometry[2] = heads
	header.DiskGeometry[3] = sectorsPerTrack

	copy(header.DiskType[:], "\x00\x00\x00\x02")
	copy(header.UniqueId[:], uuid[:])

	header.recalculateChecksum()
	return header
}

func (h *VHDFixedHeader) recalculateChecksum() {
	copy(h.Checksum[:], "\x00\x00\x00\x00")
	var hdr [vhdFixedHeaderSize]byte
	copy(hdr[:8], h.Cookie[:])
	copy(hdr[8:12], h.Features[:])
	copy(hdr[12:16], h.FileFormatVersion[:])
	copy(hdr[16:24], h.DataOffset[:])
	copy(hdr[24:28], h.Timestamp[:])
	copy(hdr[28:32], h.CreatorApplication[:])
	copy(hdr[32:36], h.CreatorVersion[:])
	copy(hdr[36:40], h.CreatorHostOS[:])
	copy(hdr[40:48], h.OriginalSize[:])
	copy(hdr[48:56], h.CurrentSize[:])
	copy(hdr[56:60], h.DiskGeometry[:])
	copy(hdr[60:64], h.DiskType[:])
	copy(hdr[64:68], h.Checksum[:])
	copy(hdr[68:84], h.UniqueId[:])
	copy(hdr[84:85], h.SavedState[:])
	copy(hdr[85:512], h.Reserved[:])

	var checksum uint32
	for i := 0; i < vhdFixedHeaderSize; i++ {
		checksum += uint32(hdr[i])
	}
	binary.BigEndian.PutUint32(h.Checksum[:], ^checksum)
}

// calculateCHS calculates the cylinder, head, sector values for the given
// total number of sectors.
// Virtual Hard Disk Image Format Specification Version 1.0
// Appendix: CHS Calculation
func calculateCHS(totalSectors uint64) (uint16, uint8, uint8) {
	var cylinders, heads, sectorsPerTrack, cylinderTimesHead uint32
	if totalSectors > 65535*16*255 {
		totalSectors = 65535 * 16 * 255
	}

	if totalSectors >= 65535*16*63 {
		sectorsPerTrack = 255
		heads = 16
		cylinderTimesHead = uint32(totalSectors / uint64(sectorsPerTrack))
	} else {
		sectorsPerTrack = 17
		cylinderTimesHead = uint32(totalSectors / uint64(sectorsPerTrack))

		heads = uint32((cylinderTimesHead + 1023) / 1024)

		if heads < 4 {
			heads = 4
		}
		if cylinderTimesHead >= (heads*1024) || heads > 16 {
			sectorsPerTrack = 31
			heads = 16
			cylinderTimesHead = uint32(totalSectors / uint64(sectorsPerTrack))
		}
		if cylinderTimesHead >= (heads * 1024) {
			sectorsPerTrack = 63
			heads = 16
			cylinderTimesHead = uint32(totalSectors / uint64(sectorsPerTrack))
		}
	}
	cylinders = cylinderTimesHead / heads
	if cylinders > 65535 || heads > 255 || sectorsPerTrack > 255 {
		panic("CHS values out of range")
	}
	return uint16(cylinders), uint8(heads), uint8(sectorsPerTrack)
}
