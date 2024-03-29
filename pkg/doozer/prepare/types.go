package prepare

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	global "github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"github.com/skatteetaten/architect/v2/pkg/docker"
	"github.com/skatteetaten/architect/v2/pkg/doozer/config"
)

// ImageMetadata utility struct containing image metadata
type ImageMetadata struct {
	BaseImage  string
	Maintainer string
	SrcPath    string
	FileName   string
	DestPath   string
	CmdScript  string
	Entrypoint string
	Labels     map[string]string
	Env        map[string]string
}

func createEnv(auroraVersion runtime.AuroraVersion, pushextratags global.PushExtraTags, imageBuildTime string) map[string]string {
	env := make(map[string]string)
	env[docker.EnvAppVersion] = string(auroraVersion.GetAppVersion())
	env[docker.EnvAuroraVersion] = auroraVersion.GetCompleteVersion()
	env[docker.EnvPushExtraTags] = pushextratags.ToStringValue()
	env[docker.TZ] = "Europe/Oslo"
	env[docker.ImageBuildTime] = imageBuildTime
	env["LANG"] = "en_US.UTF-8"
	env["HOME"] = "/u01"

	if auroraVersion.Snapshot {
		env[docker.EnvSnapshotVersion] = auroraVersion.GetGivenVersion()
	}

	return env
}

func createLabels(meta config.DeliverableMetadata) map[string]string {
	var labels = make(map[string]string)

	for k, v := range meta.Docker.Labels {
		labels[k] = v
	}

	return labels
}

func verifyMetadata(meta config.DeliverableMetadata) error {
	if meta.Docker == nil {
		return errors.Errorf("Deliverable metadata does not contain \"Docker\" element")
	} else if meta.Docker.Maintainer == "" {
		return errors.Errorf("Deliverable metadata does not contain \"Docker.Maintainer\" element")
	}
	if meta.Doozer == nil {
		return errors.Errorf("Deliverable metadata does not contain \"Doozer\" element")
	}

	return nil
}

// ReadMetadata Read and return ImageMetadata
func ReadMetadata(dockerSpec global.DockerSpec, auroraVersion runtime.AuroraVersion, meta config.DeliverableMetadata,
	baseImage runtime.DockerImage, imageBuildTime string, destinationPath string) (*ImageMetadata, error) {

	if err := verifyMetadata(meta); err != nil {
		return nil, err
	}
	env := createEnv(auroraVersion, dockerSpec.PushExtraTags, imageBuildTime)

	var destPath string

	if meta.Doozer.DestPath != "" {
		destPath = meta.Doozer.DestPath
		if meta.Doozer.DestPath != "" {
			logrus.Warnf("The destination path is overridden by provided metadata: %s", meta.Doozer.DestPath)
		}
		logrus.Debugf("Using destination path %s  from metadata", destinationPath)
	} else {
		destPath = destinationPath
	}

	return &ImageMetadata{
		BaseImage:  baseImage.GetCompleteDockerTagName(),
		Maintainer: meta.Docker.Maintainer,
		SrcPath:    meta.Doozer.SrcPath,
		FileName:   meta.Doozer.FileName,
		DestPath:   destPath,
		CmdScript:  meta.Doozer.CmdScript,
		Entrypoint: meta.Doozer.Entrypoint,
		Labels:     createLabels(meta),
		Env:        env,
	}, nil

}
