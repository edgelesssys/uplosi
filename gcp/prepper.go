/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/

package gcp

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Prepper struct{}

func (p *Prepper) Prepare(_ context.Context, imagePath, tmpDir string) (string, error) {
	// GCP images need to be packed as tar (with the oldgnu format) and compressed with gzip.
	// See https://cloud.google.com/compute/docs/import/import-existing-image#requirements_for_the_image_file
	// for details.

	rawImage, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer rawImage.Close()
	tarGzName := filepath.Join(tmpDir, "disk.tar.gz")
	outFile, err := os.Create(tarGzName)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	if err := writeTarGz(rawImage, outFile); err != nil {
		return "", fmt.Errorf("writing tar.gz: %w", err)
	}

	return tarGzName, nil
}

func writeTarGz(rawImage io.ReadSeeker, out io.Writer) error {
	rawImageSize, err := rawImage.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	if _, err := rawImage.Seek(0, io.SeekStart); err != nil {
		return err
	}

	gzipW := gzip.NewWriter(out)
	defer gzipW.Close()
	tarW := tar.NewWriter(gzipW)
	defer tarW.Close()

	if err := tarW.WriteHeader(&tar.Header{
		Name:   "disk.raw",
		Size:   rawImageSize,
		Mode:   0o644,
		Format: tar.FormatGNU,
	}); err != nil {
		return err
	}
	if _, err := io.Copy(tarW, rawImage); err != nil {
		return err
	}
	return nil
}
