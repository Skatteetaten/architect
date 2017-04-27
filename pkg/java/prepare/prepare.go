package prepare

import (
	"bufio"
	"fmt"
	"github.com/skatteetaten/architect/pkg/java/config"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"github.com/skatteetaten/architect/pkg/java/prepare/resources"
	global "github.com/skatteetaten/architect/pkg/config"
	"github.com/pkg/errors"
)

type FileGenerator interface {
	Write(writer io.Writer) error
}

func Prepare(config global.Config, buildinfo global.BuildInfo, deliverable global.Deliverable) (string, error) {

	// Create docker build folder
	dockerBuildPath, err := ioutil.TempDir("", "deliverable")

	if err != nil {
		return "", errors.Wrap(err,"Failed to create root folder")
	}

	// Unzip deliverable
	applicationPath, err := extractDeliverable(dockerBuildPath, deliverable.Path)

	if err != nil {
		return "", errors.Wrap(err,"Failed to unzip deliverable")
	}

	// Load metadata
	meta, err := loadDeliverableMetadata(filepath.Join(applicationPath, DeliveryMetadataPath))

	if err != nil {
		return "", errors.Wrap(err,"Failed to load deliverable metadata")
	}

	// Prepare application
	if err := PrepareApplication(applicationPath, meta); err != nil {
		return "", errors.Wrap(err,"Failed to prepare application")
	}

	// Dockerfile
	if err = addDockerfile(dockerBuildPath, meta, buildinfo, config); err != nil {
		return "", errors.Wrap(err,"Failed to create dockerfile")
	}

	// Runtime scripts
	if err := addRuntimeScripts(dockerBuildPath); err != nil {
		return "", errors.Wrap(err, "Failed to add scripts")
	}

	return dockerBuildPath, nil
}

func extractDeliverable(dockerBuildPath string, deliverablePath string) (string, error) {
	applicationBase := filepath.Join(dockerBuildPath, "app")
	applicationPath := filepath.Join(applicationBase, "application")

	if err := os.MkdirAll(applicationBase, 0755); err != nil {
		return "", errors.Wrap(err, "Failed to create directory")
	}

	if err := ExtractDeliverable(deliverablePath, filepath.Join(dockerBuildPath, "app")); err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("Failed to unzip deliverable %s", deliverablePath))
	}

	if err := renameApplicationDir(applicationBase); err != nil {
		return "", errors.Wrap(err, "Failed to rename application directory")
	}

	return applicationPath, nil
}

func renameApplicationDir(base string) error {
	list, err := ioutil.ReadDir(base)

	if err != nil {
		return errors.Wrap(err, "Failed to open application directory")
	} else if len(list) == 0 {
		return errors.Errorf("Root folder does not exist %s", base)
	}

	return os.Rename(filepath.Join(base, list[0].Name()), filepath.Join(base, "application"))
}

func addDockerfile(basedirPath string, meta *config.DeliverableMetadata, buildinfo global.BuildInfo, config global.Config) error {

	dockerfile := NewDockerfile(meta, buildinfo, config)

	if err := WriteFile(filepath.Join(basedirPath, "Dockerfile"), dockerfile); err != nil {
		return errors.Wrap(err, "Failed to write Dockerfile")
	}

	return nil
}

func loadDeliverableMetadata(path string) (*config.DeliverableMetadata, error) {
	fileExists, err := Exists(path)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to load deliverable metadata")
	} else if !fileExists {
		return nil, nil
	}

	reader, err := os.Open(path)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to load deliverable metadata")
	}

	deliverableMetadata, err := config.NewDeliverableMetadata(reader)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to load deliverable metadata")
	}

	return deliverableMetadata, nil
}

func addRuntimeScripts(dockerBuildPath string) error {
	scriptDirPath := filepath.Join(dockerBuildPath, "app", "bin")

	if err := os.MkdirAll(scriptDirPath, 0755); err != nil {
		return errors.Wrap(err, "Failed to create folder")
	}

	for _, asset := range resources.AssetNames() {
		bytes := resources.MustAsset(asset)
		err := ioutil.WriteFile(filepath.Join(scriptDirPath, asset), bytes, 0755)
		if err != nil {
			return errors.Wrap(err, "Failed to create script")
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

	return writer.Flush()
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
