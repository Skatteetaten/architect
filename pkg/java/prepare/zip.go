package prepare

/**
Source: http://blog.ralch.com/tutorial/golang-working-with-zip/
*/

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func Unzip(extractedDirPath string, archivePath string) error {

	zipReader, err := zip.OpenReader(archivePath)

	if err != nil {
		return err
	}

	defer zipReader.Close()

	for _, zipEntry := range zipReader.File {

		extractedPath := filepath.Join(extractedDirPath, zipEntry.Name)

		if zipEntry.FileInfo().Mode()&os.ModeDir != 0 {
			err = handleDirectory(extractedPath, zipEntry)
		} else if zipEntry.FileInfo().Mode()&os.ModeSymlink != 0 {
			err = handleSymlink(extractedPath, zipEntry)
		} else {
			err = handleRegular(extractedPath, zipEntry)
		}

		if err != nil {
			return 	fmt.Errorf("Failed to extract %s: %v", zipEntry.Name, err)
		}
	}

	return nil
}

func handleDirectory(extractedPath string, zipEntry *zip.File) error {
	return os.MkdirAll(extractedPath, zipEntry.Mode())
}

func handleSymlink(extractedPath string, zipEntry *zip.File) error {
	sourceFile, err := zipEntry.Open()

	if err != nil {
		return err
	}

	defer sourceFile.Close()

	size := zipEntry.FileInfo().Size()
	buffer := make([]byte, size)

	_, err = sourceFile.Read(buffer)

	if err != nil {
		return err
	}

	targetPath := string(buffer[:size])

	return os.Symlink(targetPath, extractedPath)
}

func handleRegular(extractedPath string, zipEntry *zip.File) error {
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
	} else if ! os.IsNotExist(err) {
		return err
	}

	return os.MkdirAll(dirPath, 0755)
}