package prepare

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	global "github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/doozer/config"
	"github.com/skatteetaten/architect/pkg/util"
	"io"
)

var dockerFileTemplateBody string = `FROM {{.BaseImage}}

MAINTAINER {{.Maintainer}}
LABEL{{range $key, $value := .Labels}} {{$key}}="{{$value}}"{{end}}

# temp env hack until standard base image is available
ENV LANG='en_US.UTF-8' \
    TZ=Europe/Oslo \
    HOME=/u01

COPY ./app radish.json $HOME/
COPY ./app/application/{{.SrcPath}}{{.FileName}} {{.DestPath}}

RUN find $HOME/application -type d -exec chmod 755 {} + && \
	find $HOME/application -type f -exec chmod 644 {} + && \
	mkdir -p $HOME/logs && \
	chmod 777 $HOME/logs && \
	ln -s $HOME/logs $HOME/application/logs

ENV{{range $key, $value := .Env}} {{$key}}="{{$value}}"{{end}}
`
var dockerFileTemplateCmd string = `CMD "{{.CmdScript}}"
`

type DockerfileData struct {
	BaseImage  string
	Maintainer string
	SrcPath    string
	FileName   string
	DestPath   string
	CmdScript  string
	Labels     map[string]string
	Env        map[string]string
}

func createEnv(auroraVersion runtime.AuroraVersion, pushextratags global.PushExtraTags, imageBuildTime string) map[string]string {
	env := make(map[string]string)
	env[docker.ENV_APP_VERSION] = string(auroraVersion.GetAppVersion())
	env[docker.ENV_AURORA_VERSION] = auroraVersion.GetCompleteVersion()
	env[docker.ENV_PUSH_EXTRA_TAGS] = pushextratags.ToStringValue()
	env[docker.TZ] = "Europe/Oslo"
	env[docker.IMAGE_BUILD_TIME] = imageBuildTime

	if auroraVersion.Snapshot {
		env[docker.ENV_SNAPSHOT_TAG] = auroraVersion.GetGivenVersion()
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

func NewDockerFile(dockerSpec global.DockerSpec, auroraVersion runtime.AuroraVersion, meta config.DeliverableMetadata,
	baseImage runtime.DockerImage, imageBuildTime string, destinationPath string) util.WriterFunc {
	return func(writer io.Writer) error {
		if err := verifyMetadata(meta); err != nil {
			return err
		}
		env := createEnv(auroraVersion, dockerSpec.PushExtraTags, imageBuildTime)

		dockerFileTemplate := dockerFileTemplateBody
		if meta.Doozer.CmdScript != "" {
			dockerFileTemplate += dockerFileTemplateCmd
		}

		var destPath string
		if destinationPath != "" {
			destPath = destinationPath
			if meta.Doozer.DestPath != "" {
				logrus.Warnf("The destination path is overridden by the base image: %s", meta.Doozer.DestPath)
			}
			logrus.Debugf("Using destination path %s from image", destinationPath)
		} else {
			destPath = meta.Doozer.DestPath
		}

		if meta.Doozer.FileName != "" {
			destPath += meta.Doozer.FileName
		}

		data := &DockerfileData{
			BaseImage:  baseImage.GetCompleteDockerTagName(),
			Maintainer: meta.Docker.Maintainer,
			SrcPath:    meta.Doozer.SrcPath,
			FileName:   meta.Doozer.FileName,
			DestPath:   destPath,
			CmdScript:  meta.Doozer.CmdScript,
			Labels:     createLabels(meta),
			Env:        env,
		}

		return util.NewTemplateWriter(data, "Dockerfile", dockerFileTemplate)(writer)
	}
}
