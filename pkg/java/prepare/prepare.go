package prepare

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

type BuildConfiguration struct {
	BuildContext string
	Env          map[string]string
	Labels       map[string]string
	Cmd          []string
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
		//TODO: Hack. Remove code later
	} else if architecture, exists := baseImage.ImageInfo.Labels["www.skatteetaten.no-imageArchitecture"]; exists && architecture == "java-test" {
		logrus.Info("Running test image build")

		if err := fileWriter(newRadishDescriptor(meta, filepath.Join(util.DockerBasedir, util.ApplicationFolder)), "radish.json"); err != nil {
			return "", errors.Wrap(err, "Unable to create radish descriptor")
		}
		if err = fileWriter(NewRadishTestImageDockerFile(dockerSpec, *auroraVersions, *meta, baseImage.DockerImage, docker.GetUtcTimestamp()),
			"Dockerfile"); err != nil {
			return "", errors.Wrap(err, "Failed to create Dockerfile")
		}
	} else {
		return "", fmt.Errorf("The base image provided does not support radish. Make sure you use the latest version")
	}

	return dockerBuildPath, nil
}

func PrepareLayers(dockerSpec config.DockerSpec, auroraVersions *runtime.AuroraVersion, deliverable nexus.Deliverable, baseImage runtime.BaseImage) (*BuildConfiguration, error) {
	buildPath, err := ioutil.TempDir("", "deliverable")

	if err != nil {
		return nil, errors.Wrap(err, "Failed to create root folder of Docker context")
	}

	if err := os.MkdirAll(buildPath+"/layer/u01", 0755); err != nil {
		return nil, errors.Wrap(err, "Failed to create layer structure")
	}

	if err := os.MkdirAll(buildPath+"/layer/u01/logs", 0777); err != nil {
		return nil, errors.Wrap(err, "Failed to create log folder")
	}

	// Unzip deliverable
	applicationRoot := filepath.Join(buildPath, util.DockerfileApplicationFolder)
	renamedApplicationFolder := filepath.Join(buildPath, "/layer/u01/application")

	if err := util.ExtractDeliverable(deliverable.Path, applicationRoot); err != nil {
		return nil, errors.Wrapf(err, "Failed to extract application archive")
	}

	if err := util.RenameSingleFolderInDirectory(applicationRoot, renamedApplicationFolder); err != nil {
		return nil, errors.Wrap(err, "Failed to rename application directory in build context")
	}

	// Load metadata
	meta, err := loadDeliverableMetadata(filepath.Join(buildPath, "layer/u01/application/metadata/openshift.json"))

	if err != nil {
		return nil, errors.Wrap(err, "Failed to read application metadata")
	}

	fileWriter := util.NewFileWriter(buildPath + "/layer/u01")

	if err := fileWriter(newRadishDescriptor(meta, filepath.Join(util.DockerBasedir, util.ApplicationFolder)), "radish.json"); err != nil {
		return nil, errors.Wrap(err, "Unable to create radish descriptor")
	}

	//Create symlink
	target := buildPath + "/layer/u01/logs/"
	err = os.Symlink(target, buildPath+"/layer/u01/application/logs")
	if err != nil {
		return nil, errors.Wrap(err, "Unable to create symlink")
	}

	return &BuildConfiguration{
		BuildContext: buildPath,
		Env:          createEnv(*auroraVersions, dockerSpec.PushExtraTags, docker.GetUtcTimestamp()),
		Labels:       createLabels(*meta),
		Cmd:          nil,
	}, nil

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
