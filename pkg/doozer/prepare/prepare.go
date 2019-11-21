package prepare

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	deliverable "github.com/skatteetaten/architect/pkg/doozer/config"
	"github.com/skatteetaten/architect/pkg/nexus"
	"github.com/skatteetaten/architect/pkg/util"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

const (
	DeliveryMetadataPath = "metadata/openshift.json"
)

type FileGenerator interface {
	Write(writer io.Writer) error
}

func Prepare(dockerSpec config.DockerSpec, auroraVersions *runtime.AuroraVersion, deliverable nexus.Deliverable, baseImage runtime.BaseImage) (string, error) {

	// Create docker build folder
	dockerBuildPath, err := ioutil.TempDir("", "deliverable")

	if err != nil {
		return "", errors.Wrap(err, "Failed to create root folder of Docker context")
	}

	// Unzip deliverable
	applicationFolder := filepath.Join(dockerBuildPath, ApplicationBuildFolder)

	err = extractAndRenameDeliverable(dockerBuildPath, deliverable.Path)

	if err != nil {
		return "", errors.Wrap(err, "Failed to extract application archive")
	}

	// Load metadata
	metadatafolder := filepath.Join(applicationFolder, DeliveryMetadataPath)
	logrus.Debugf("metadatafolder: %v", metadatafolder)
	meta, err := loadDeliverableMetadata(metadatafolder)
	logrus.Debugf("meta: %v", meta)
	if err != nil {
		return "", errors.Wrap(err, "Failed to read application metadata")
	}

	fileWriter := util.NewFileWriter(dockerBuildPath)

	logrus.Info("Running radish build (doozer)")
	if err := fileWriter(newRadishDescriptor(meta, filepath.Join(DockerBasedir, ApplicationFolder)), "radish.json"); err != nil {
		return "", errors.Wrap(err, "Unable to create radish descriptor")
	}
	if err = fileWriter(NewDockerFile(dockerSpec, *auroraVersions, *meta, baseImage.DockerImage, docker.GetUtcTimestamp()),
		"Dockerfile"); err != nil {
		return "", errors.Wrap(err, "Failed to create Dockerfile")
	}

	return dockerBuildPath, nil
}

func extractAndRenameDeliverable(dockerBuildFolder string, deliverablePath string) error {

	applicationRoot := filepath.Join(dockerBuildFolder, DockerfileApplicationFolder)
	renamedApplicationFolder := filepath.Join(dockerBuildFolder, ApplicationBuildFolder)
	if err := os.MkdirAll(dockerBuildFolder, 0755); err != nil {
		return errors.Wrap(err, "Failed to create application directory in Docker context")
	}

	if err := ExtractDeliverable(deliverablePath, applicationRoot); err != nil {
		return errors.Wrapf(err, "Failed to extract application archive")
	}

	if err := renameSingleFolderInDirectory(applicationRoot, renamedApplicationFolder); err != nil {
		return errors.Wrap(err, "Failed to rename application directory in Docker context")
	} else {
		return nil
	}
}

// When we unzip the delivery, it will have an additional level.
// eg. app/myapplication-LEVERANSEPAKKE-SNAPSHOT -> app/application
func renameSingleFolderInDirectory(base string, newName string) error {
	list, err := ioutil.ReadDir(base)

	if err != nil {
		return errors.Wrapf(err, "Failed to open application directory %s", base)
	} else if len(list) == 0 {
		return errors.Errorf("Application root folder does not exist %s", base)
	}

	folderToBeRenamed := filepath.Join(base, list[0].Name())
	if err = os.Rename(folderToBeRenamed, newName); err != nil {
		return errors.Wrapf(err, "Rename from %s to %s failed", folderToBeRenamed, newName)
	} else {
		return nil
	}
}

func loadDeliverableMetadata(metafile string) (*deliverable.DeliverableMetadata, error) {
	fileExists, err := Exists(metafile)

	if err != nil {
		return nil, errors.Wrapf(err, "Could not find %s in deliverable", path.Base(metafile))
	} else if !fileExists {
		return nil, errors.Errorf("Could not find %s in deliverable", path.Base(metafile))
	}

	reader, err := os.Open(metafile)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to open application metadata file")
	}

	deliverableMetadata, err := deliverable.NewDeliverableMetadata(reader)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to load metadata from %s", path.Base(metafile))
	}

	return deliverableMetadata, nil
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
