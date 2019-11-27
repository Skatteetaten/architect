package prepare

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	deliverable "github.com/skatteetaten/architect/pkg/java/config"
	"github.com/skatteetaten/architect/pkg/nexus"
	"github.com/skatteetaten/architect/pkg/util"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
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
	applicationFolder := filepath.Join(dockerBuildPath, util.ApplicationBuildFolder)
	err = util.ExtractAndRenameDeliverable(dockerBuildPath, deliverable.Path)

	if err != nil {
		return "", errors.Wrap(err, "Failed to extract application archive")
	}

	// Load metadata
	meta, err := loadDeliverableMetadata(filepath.Join(applicationFolder, util.DeliveryMetadataPath))

	if err != nil {
		return "", errors.Wrap(err, "Failed to read application metadata")
	}

	fileWriter := util.NewFileWriter(dockerBuildPath)

	if architecture, exists := baseImage.ImageInfo.Labels["www.skatteetaten.no-imageArchitecture"]; exists && architecture == "java" {

		logrus.Info("Running radish build")
		if err := fileWriter(newRadishDescriptor(meta, filepath.Join(util.DockerBasedir, util.ApplicationFolder)), "radish.json"); err != nil {
			return "", errors.Wrap(err, "Unable to create radish descriptor")
		}
		if err = fileWriter(NewRadishDockerFile(dockerSpec, *auroraVersions, *meta, baseImage.DockerImage, docker.GetUtcTimestamp()),
			"Dockerfile"); err != nil {
			return "", errors.Wrap(err, "Failed to create Dockerfile")
		}
	} else {
		return "", fmt.Errorf("The base image provided does not support radish. Make sure you use the latest version")
	}

	return dockerBuildPath, nil
}

func loadDeliverableMetadata(metafile string) (*deliverable.DeliverableMetadata, error) {
	fileExists, err := util.Exists(metafile)

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
