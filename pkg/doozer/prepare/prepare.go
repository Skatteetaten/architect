package prepare

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"github.com/skatteetaten/architect/v2/pkg/docker"
	deliverable "github.com/skatteetaten/architect/v2/pkg/doozer/config"
	"github.com/skatteetaten/architect/v2/pkg/nexus"
	process "github.com/skatteetaten/architect/v2/pkg/process/build"
	"github.com/skatteetaten/architect/v2/pkg/util"
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
	EntryPoint   []string
}

const (
	DeliveryMetadataPath = "metadata/openshift.json"
)

func Prepper() process.Prepper {
	return func(cfg *config.Config, auroraVersion *runtime.AuroraVersion, deliverable nexus.Deliverable,
		baseImage runtime.BaseImage) ([]docker.BuildConfig, error) {

		logrus.Debug("Pull output image")
		buildContext, err := prepareLayers(cfg.DockerSpec, auroraVersion, deliverable, baseImage)

		if err != nil {
			return nil, errors.Wrap(err, "Error prepare artifact")
		}

		buildConf := docker.BuildConfig{
			AuroraVersion:    auroraVersion,
			BuildFolder:      buildContext.BuildContext,
			DockerRepository: cfg.DockerSpec.OutputRepository,
			Image:            baseImage.DockerImage,
			Env:              buildContext.Env,
			Labels:           buildContext.Labels,
			Cmd:              buildContext.Cmd,
			Entrypoint:       buildContext.EntryPoint,
		}
		return []docker.BuildConfig{buildConf}, nil
	}
}

func prepareLayers(dockerSpec config.DockerSpec, auroraVersions *runtime.AuroraVersion, deliverable nexus.Deliverable, baseImage runtime.BaseImage) (*BuildConfiguration, error) {
	//Create build context
	buildContext, err := ioutil.TempDir("", "deliverable")
	filewriter := util.NewFileWriter(buildContext)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to create root folder for build context")
	}

	if err := os.MkdirAll(buildContext+"/layer/u01/application", 0755); err != nil {
		return nil, errors.Wrap(err, "Failed to create layer structure")
	}

	if err := os.MkdirAll(buildContext+"/layer/u01/logs", 0777); err != nil {
		return nil, errors.Wrap(err, "Failed to create log folder")
	}

	// Unzip deliverable
	applicationFolder := filepath.Join(buildContext, util.ApplicationBuildFolder)
	err = util.ExtractAndRenameDeliverable(buildContext, deliverable.Path)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to extract application archive")
	}

	// Load metadata
	metadatafolder := filepath.Join(applicationFolder, DeliveryMetadataPath)
	logrus.Debugf("metadatafolder: %v", metadatafolder)
	deliverableMetadata, err := loadDeliverableMetadata(metadatafolder)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read application metadata")
	}

	logrus.Info("Running radish build (doozer)")
	if err := filewriter(newRadishDescriptor(deliverableMetadata, filepath.Join(buildContext, "/layer/u01/")), "radish.json"); err != nil {
		return nil, errors.Wrap(err, "Unable to create radish descriptor")
	}

	destinationPath := baseImage.ImageInfo.Labels["www.skatteetaten.no-destinationPath"]

	imageMetadata, err := ReadMetadata(dockerSpec, *auroraVersions, *deliverableMetadata, baseImage.DockerImage, docker.GetUtcTimestamp(), destinationPath)
	if err != nil {
		return nil, errors.Wrap(err, "Could not read image metadata")
	}

	var cmd []string
	if imageMetadata.CmdScript != "" {
		cmd = strings.Split(imageMetadata.CmdScript, " ")
	}
	var entrypoint []string
	if imageMetadata.Entrypoint != "" {
		entrypoint = strings.Split(imageMetadata.Entrypoint, " ")
	}

	fileinfo, err := os.Stat(filepath.Join(buildContext, "app/application", strings.TrimSpace(imageMetadata.SrcPath), strings.TrimSpace(imageMetadata.FileName)))

	if err == nil && fileinfo.IsDir() {
		src := filepath.Join(buildContext, "app/application", imageMetadata.SrcPath)
		dst := filepath.Join(buildContext, "layer", imageMetadata.DestPath)
		if err := util.CopyDirectory(src, dst); err != nil {
			return nil, errors.Wrapf(err, "Could not copy directory from src=%s to dst=%s", src, dst)
		}
		if len(cmd) > 0 && len(entrypoint) == 0 {
			executable := filepath.Join(buildContext, "app/application", "layer", cmd[0])
			err = os.Chmod(executable, 0755)
			if err != nil {
				return nil, errors.Wrap(err, "Could not set permission")
			}
		}
	} else {
		srcFile := filepath.Join(buildContext, "app/application", strings.TrimSpace(imageMetadata.SrcPath), strings.TrimSpace(imageMetadata.FileName))
		dstPath := filepath.Join(buildContext, "layer", imageMetadata.DestPath)
		err = os.MkdirAll(dstPath, 0755)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not create destination path: %s", dstPath)
		}
		dstFile := filepath.Join(dstPath, imageMetadata.FileName)

		if err := util.Copy(srcFile, dstFile); err != nil {
			return nil, errors.Wrapf(err, "Could not copy file %s to dst=%s", imageMetadata.FileName, dstFile)
		}

		err = os.Chmod(dstFile, 0755)
		if err != nil {
			return nil, errors.Wrap(err, "Could not set permission")
		}
	}

	//Create symlink
	target := buildContext + "/layer/u01/logs/"
	err = os.Symlink(target, buildContext+"/layer/u01/application/logs")
	if err != nil {
		return nil, errors.Wrap(err, "Unable to create symlink")
	}

	return &BuildConfiguration{
		BuildContext: buildContext,
		Env:          imageMetadata.Env,
		Labels:       imageMetadata.Labels,
		Cmd:          cmd,
		EntryPoint:   entrypoint,
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
