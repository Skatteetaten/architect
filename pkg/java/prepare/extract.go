package prepare

import (
	"archive/zip"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
)

func ExtractDeliverable(archivePath string, extractedDirPath string) error {

	zipReader, err := zip.OpenReader(archivePath)

	if err != nil {
		return errors.Wrapf(err, "Failed to open archive %s", archivePath)
	}

	defer zipReader.Close()

	for _, zipEntry := range zipReader.File {

		extractedPath := filepath.Join(extractedDirPath, zipEntry.Name)

		if zipEntry.FileInfo().IsDir() {
			if err = extractDirectory(extractedPath, zipEntry); err != nil {
				return errors.Wrapf(err, "Failed to extract directory  %s", zipEntry.Name)
			}
		} else {
			if err = extractRegular(extractedPath, zipEntry); err != nil {
				return errors.Wrapf(err, "Failed to extract file %s", zipEntry.Name)
			}
		}
	}

	return nil
}

func extractDirectory(extractedPath string, zipEntry *zip.File) error {
	return os.MkdirAll(extractedPath, zipEntry.Mode())
}

func extractRegular(extractedPath string, zipEntry *zip.File) error {
	sourceFile, err := zipEntry.Open()

	if err != nil {
		return errors.Wrapf(err, "Failed to open file  %s", zipEntry.Name)
	}

	defer sourceFile.Close()

	// File may have been added to archive before all parent entries.
	err = fillPathGap(extractedPath)

	if err != nil {
		return errors.Errorf("Failed to create directory tree for %s", zipEntry.Name)
	}

	extractedFile, err := os.Create(extractedPath)

	if err != nil {
		return errors.Wrap(err, "Failed to create file")
	}

	defer extractedFile.Close()

	err = extractedFile.Chmod(zipEntry.FileInfo().Mode())

	if err != nil {
		return errors.Wrap(err, "Failed to change file permission")
	}

	_, err = io.Copy(extractedFile, sourceFile)

	return err

}

func fillPathGap(path string) error {
	dirPath := filepath.Dir(path)

	_, err := os.Stat(dirPath)

	if err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "Failed to check if directory %s exists", dirPath)
	}

	if err = os.MkdirAll(dirPath, 0755); err != nil {
		return errors.Wrapf(err, "Failed to create directory")
	}

	return nil
}
