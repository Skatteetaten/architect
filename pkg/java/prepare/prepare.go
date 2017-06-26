package prepare

import (
	"bufio"
	"github.com/skatteetaten/architect/pkg/java/config"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"github.com/skatteetaten/architect/pkg/java/prepare/resources"
	global "github.com/skatteetaten/architect/pkg/config"
	"github.com/pkg/errors"
	"path"
	"os/user"
)

type FileGenerator interface {
	Write(writer io.Writer) error
}

func Prepare(buildinfo global.BuildInfo, deliverable global.Deliverable) (string, error) {

	// Create docker build folder
	dockerBuildPath, err := ioutil.TempDir("", "deliverable")

	if err != nil {
		return "", errors.Wrap(err,"Failed to create root folder of Docker context")
	}

	// Unzip deliverable
	applicationPath, applicationBase, err := extractDeliverable(dockerBuildPath, deliverable.Path)

	if err != nil {
		return "", errors.Wrap(err,"Failed to extract application archive")
	}

	// Load metadata
	meta, err := loadDeliverableMetadata(filepath.Join(applicationPath, DeliveryMetadataPath))

	if err != nil {
		return "", errors.Wrap(err,"Failed to read application metadata")
	}

	// Runtime scripts
	if err := addRuntimeScripts(dockerBuildPath); err != nil {
		return "", errors.Wrap(err, "Failed to add static content to Docker context")
	}

	// Prepare application
	if err := PrepareApplication(applicationPath, applicationBase, meta); err != nil {
		return "", errors.Wrap(err,"Failed to prepare application")
	}

	// Dockerfile
	if err = addDockerfile(dockerBuildPath, meta, buildinfo); err != nil {
		return "", errors.Wrap(err,"Failed to create Dockerfile")
	}

	return dockerBuildPath, nil
}

func extractDeliverable(dockerBuildPath string, deliverablePath string) (string, string, error) {
	applicationBase := filepath.Join(dockerBuildPath, ApplicationDir)

	if err := os.MkdirAll(applicationBase, 0755); err != nil {
		return "", "", errors.Wrap(err, "Failed to create application directory in Docker context")
	}

	if err := ExtractDeliverable(deliverablePath, filepath.Join(dockerBuildPath, "app")); err != nil {
		return "", "", errors.Wrapf(err, "Failed to extract application archive")
	}

	if applicationPath, err := renameApplicationDir(applicationBase); err != nil {
		return "", "", errors.Wrap(err, "Failed to rename application directory in Docker context")
	} else {
		return applicationPath, applicationBase, nil
	}
}

func renameApplicationDir(base string) (string, error) {
	list, err := ioutil.ReadDir(base)

	if err != nil {
		return "", errors.Wrapf(err, "Failed to open application directory %s", base)
	} else if len(list) == 0 {
		return "", errors.Errorf("Application root folder does not exist %s", base)
	}

	renamedApplicationPath := filepath.Join(base, "application")
	if err = os.Rename(filepath.Join(base, list[0].Name()), renamedApplicationPath); err != nil {
		return "", errors.Wrap(err, "Rename of application path to application failed")
	} else {
		return renamedApplicationPath, nil;
	}
}

func addDockerfile(basedirPath string, meta *config.DeliverableMetadata, buildinfo global.BuildInfo) error {

	dockerfile, err := NewDockerfile(meta, buildinfo)

	if err != nil {
		return errors.Wrap(err, "Failed to prepare Dockerfile")
	}

	if err := WriteFile(filepath.Join(basedirPath, "Dockerfile"), dockerfile); err != nil {
		return errors.Wrap(err, "Failed to write Dockerfile")
	}

	return nil
}

func loadDeliverableMetadata(metafile string) (*config.DeliverableMetadata, error) {
	fileExists, err := Exists(metafile)

	if err != nil {
		return nil, errors.Wrapf(err, "Could not find %s in deliverable", path.Base(metafile))
	} else if ! fileExists {
		return nil, errors.Errorf("Could not find %s in deliverable", path.Base(metafile))
	}

	reader, err := os.Open(metafile)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to open application metadata file")
	}

	deliverableMetadata, err := config.NewDeliverableMetadata(reader)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to load metadata from %s", path.Base(metafile))
	}

	return deliverableMetadata, nil
}

func addRuntimeScripts(dockerBuildPath string) error {
	scriptDirPath := filepath.Join(dockerBuildPath, "app", "bin")

	if err := os.MkdirAll(scriptDirPath, 0755); err != nil {
		return errors.Wrap(err, "Failed to create resource folder")
	}

	for _, asset := range resources.AssetNames() {
		bytes := resources.MustAsset(asset)
		err := ioutil.WriteFile(filepath.Join(scriptDirPath, asset), bytes, 0755)
		if err != nil {
			return errors.Wrapf(err, "Failed add resource %s", asset)
		}
	}

	return nil
}

func WriteFile(path string, generator FileGenerator) error {

	file, err := os.Create(path)

	if err != nil {
		return errors.Wrap(err, "Failed to create file")
	}

	defer file.Close()

	writer := bufio.NewWriter(file)

	err = generator.Write(writer)

	if err != nil {
		return errors.Wrap(err, "Failed to write file")
	}

	if err = writer.Flush(); err != nil {
		return errors.Wrap(err, "Failed to flush file content")
	}

	return nil
}

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)

	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, errors.Wrap(err, "Failed to stat file")
	}

	return true, nil
}
