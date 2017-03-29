package prepare

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func UnzipDeliverable(archivePath string, extractedDirPath string) error {

	zipReader, err := zip.OpenReader(archivePath)

	if err != nil {
		return err
	}

	defer zipReader.Close()

	for _, zipEntry := range zipReader.File {

		extractedPath := filepath.Join(extractedDirPath, zipEntry.Name)

		if zipEntry.FileInfo().IsDir() {
			err = extractDirectory(extractedPath, zipEntry)
		} else {
			err = extractRegular(extractedPath, zipEntry)
		}

		if err != nil {
			return fmt.Errorf("Failed to extract %s: %v", zipEntry.Name, err)
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
		return err
	}

	defer sourceFile.Close()

	// File may have been added to archive before all parent entries.
	err = fillPathGap(extractedPath)

	if err != nil {
		return err
	}

	extractedFile, err := os.Create(extractedPath)

	if err != nil {
		return err
	}

	defer extractedFile.Close()

	err = extractedFile.Chmod(zipEntry.FileInfo().Mode())

	if err != nil {
		return err
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
		return err
	}

	return os.MkdirAll(dirPath, 0755)
}
