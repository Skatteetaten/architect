package prepare

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	applicationFolder := filepath.Join(dockerBuildPath, util.ApplicationBuildFolder)
	err = util.ExtractAndRenameDeliverable(dockerBuildPath, deliverable.Path)

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
	if err := fileWriter(newRadishDescriptor(meta, filepath.Join(util.DockerBasedir, util.ApplicationFolder)), "radish.json"); err != nil {
		return "", errors.Wrap(err, "Unable to create radish descriptor")
	}
	if err = fileWriter(NewDockerFile(dockerSpec, *auroraVersions, *meta, baseImage.DockerImage, docker.GetUtcTimestamp()),
		"Dockerfile"); err != nil {
		return "", errors.Wrap(err, "Failed to create Dockerfile")
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
