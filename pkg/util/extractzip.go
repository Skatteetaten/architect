package util

import (
	"archive/zip"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	// The base directory where all code is copied in the Docker image
	DockerBasedir = "/u01"
	// Where in the build folder the application is put
	DockerfileApplicationFolder = "app"
	ApplicationFolder           = "application"
	LayerFolder                 = "layer"
	// The directory where the application is prepared
	ApplicationBuildFolder = DockerfileApplicationFolder + "/" + ApplicationFolder
	ApplicationLayerFolder = DockerBasedir + "/" + ApplicationFolder
	DeliveryMetadataPath   = "metadata/openshift.json"
)

//ExtractAndRenameDeliverable extract and rename
func ExtractAndRenameDeliverable(dockerBuildFolder string, deliverablePath string) error {

	applicationRoot := filepath.Join(dockerBuildFolder, DockerfileApplicationFolder)
	renamedApplicationFolder := filepath.Join(dockerBuildFolder, ApplicationBuildFolder)
	if err := MkdirAllWithPermissions(dockerBuildFolder, 0755); err != nil {
		return errors.Wrap(err, "Failed to create application directory in Docker context")
	}

	if err := ExtractDeliverable(deliverablePath, applicationRoot); err != nil {
		return errors.Wrapf(err, "Failed to extract application archive")
	}

	if err := RenameSingleFolderInDirectory(applicationRoot, renamedApplicationFolder); err != nil {
		return errors.Wrap(err, "Failed to rename application directory in Docker context")
	}
	return nil

}

// ExtractDeliverable extract archive to dest
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
	return MkdirAllWithPermissions(extractedPath, zipEntry.Mode())
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

	if err = MkdirAllWithPermissions(dirPath, 0755); err != nil {
		return errors.Wrapf(err, "Failed to create directory")
	}

	return nil
}

// When we unzip the delivery, it will have an additional level.
// eg. app/myapplication-LEVERANSEPAKKE-SNAPSHOT -> app/application
func RenameSingleFolderInDirectory(base string, newName string) error {
	list, err := ioutil.ReadDir(base)

	if err != nil {
		return errors.Wrapf(err, "Failed to open application directory %s", base)
	} else if len(list) == 0 {
		return errors.Errorf("Application root folder does not exist %s", base)
	}

	folderToBeRenamed := filepath.Join(base, list[0].Name())
	if err = os.Rename(folderToBeRenamed, newName); err != nil {
		return errors.Wrapf(err, "Rename from %s to %s failed", folderToBeRenamed, newName)
	}
	return nil
}
