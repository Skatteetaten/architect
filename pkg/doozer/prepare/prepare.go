package prepare

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	deliverable "github.com/skatteetaten/architect/pkg/doozer/config"
	"github.com/skatteetaten/architect/pkg/nexus"
	process "github.com/skatteetaten/architect/pkg/process/build"
	"github.com/skatteetaten/architect/pkg/util"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
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

		if strings.ToLower(cfg.BuildStrategy) == config.Layer {
			return nil, errors.New("Doozer layer build not supported")
		}

		logrus.Debug("Pull output image")
		buildContext, err := PrepareLayers(cfg.DockerSpec, auroraVersion, deliverable, baseImage)

		if err != nil {
			return nil, errors.Wrap(err, "Error prepare artifact")
		}

		buildConf := docker.DockerBuildConfig{
			AuroraVersion:    auroraVersion,
			BuildFolder:      buildContext.BuildContext,
			DockerRepository: cfg.DockerSpec.OutputRepository,
			Baseimage:        baseImage.DockerImage,
			Env:              buildContext.Env,
			Labels:           buildContext.Labels,
			Cmd:              buildContext.Cmd,
		}
		return []docker.DockerBuildConfig{buildConf}, nil
	}
}

//TODO: Lag testdata til denne og skriv en test
func PrepareLayers(dockerSpec config.DockerSpec, auroraVersions *runtime.AuroraVersion, deliverable nexus.Deliverable, baseImage runtime.BaseImage) (*BuildConfiguration, error) {
	//Create build context
	buildContext, err := ioutil.TempDir("", "deliverable")
	fileWriter := util.NewFileWriter(buildContext)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to create root folder for build context")
	}

	if err := os.MkdirAll(buildContext+"/layer/u01", 0755); err != nil {
		return nil, errors.Wrap(err, "Failed to create layer structure")
	}

	if err := os.MkdirAll(buildContext+"/layer/u01/logs", 0777); err != nil {
		return nil, errors.Wrap(err, "Failed to create log folder")
	}

	// Load metadata
	deliverableMetadata, err := loadDeliverableMetadata(filepath.Join(buildContext, "layer/u01/application/metadata/openshift.json"))

	logrus.Info("Running radish build (doozer)")
	if err := fileWriter(newRadishDescriptor(deliverableMetadata, buildContext+"/layer/u01/radish.json")); err != nil {
		return nil, errors.Wrap(err, "Unable to create radish descriptor")
	}

	destinationPath := baseImage.ImageInfo.Labels["www.skatteetaten.no-destinationPath"]

	imageMetadata, err := ReadMetadata(dockerSpec, *auroraVersions, *deliverableMetadata, baseImage.DockerImage, docker.GetUtcTimestamp(), destinationPath)
	if err != nil {
		return nil, errors.Wrap(err, "Could not read image metadata")
	}

	// Unzip deliverable
	applicationRoot := filepath.Join(buildContext, util.DockerfileApplicationFolder)

	if err := util.ExtractDeliverable(deliverable.Path, applicationRoot); err != nil {
		return nil, errors.Wrapf(err, "Failed to extract application archive")
	}

	fileinfo, err := os.Stat(buildContext + "/" + imageMetadata.SrcPath)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get fileinfo of file %s", imageMetadata.SrcPath)
	}

	if fileinfo.IsDir() {
		src := filepath.Join(buildContext, imageMetadata.SrcPath)
		dst := filepath.Join(buildContext, imageMetadata.DestPath)
		if err := util.CopyDirectory(src, dst); err != nil {
			return nil, errors.Wrapf(err, "Could not copy directory from src=%s to dst=%s", src, dst)
		}
	} else {
		srcFile := filepath.Join(buildContext, imageMetadata.SrcPath)
		dstFile := filepath.Join(buildContext, imageMetadata.DestPath, imageMetadata.FileName)
		if err := util.Copy(srcFile, dstFile); err != nil {
			return nil, errors.Wrapf(err, "Could not copy file %s to dst=%s", imageMetadata.FileName, imageMetadata.DestPath)
		}
	}

	//Create symlink
	target := buildContext + "/layer/u01/logs/"
	err = os.Symlink(target, buildContext+"/layer/u01/application/logs")
	if err != nil {
		return nil, errors.Wrap(err, "Unable to create symlink")
	}

	var cmd []string
	if imageMetadata.CmdScript == "" {
		cmd = nil
	} else {
		cmd = []string{imageMetadata.CmdScript}
	}

	return &BuildConfiguration{
		BuildContext: buildContext,
		Env:          imageMetadata.Env,
		Labels:       imageMetadata.Labels,
		Cmd:          cmd,
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
