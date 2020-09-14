package prepare

import (
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	deliverable "github.com/skatteetaten/architect/pkg/java/config"
	"github.com/skatteetaten/architect/pkg/nexus"
	process "github.com/skatteetaten/architect/pkg/process/build"
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

func Prepper() process.Prepper {
	return func(cfg *config.Config, auroraVersion *runtime.AuroraVersion, deliverable nexus.Deliverable,
		baseImage runtime.BaseImage) ([]docker.DockerBuildConfig, error) {
		buildConfiguration, err := prepareLayers(cfg.DockerSpec, auroraVersion, deliverable)
		if err != nil {
			return nil, errors.Wrap(err, "Error while preparing layers")
		}
		return []docker.DockerBuildConfig{{
			AuroraVersion:    auroraVersion,
			DockerRepository: cfg.DockerSpec.OutputRepository,
			BuildFolder:      buildConfiguration.BuildContext,
			Baseimage:        baseImage.DockerImage,
			Env:              buildConfiguration.Env,
			Labels:           buildConfiguration.Labels,
			Cmd:              buildConfiguration.Cmd,
		}}, nil
	}
}

func prepareLayers(dockerSpec config.DockerSpec, auroraVersions *runtime.AuroraVersion, deliverable nexus.Deliverable) (*BuildConfiguration, error) {
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
	fileExists := util.Exists(metafile)

	if !fileExists {
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
