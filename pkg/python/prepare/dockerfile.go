package prepare

import (
	"github.com/pkg/errors"
	global "github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/python/config"
	"github.com/skatteetaten/architect/pkg/util"
	"io"
)

var dockerFileTemplateBody = `FROM {{.BaseImage}}

MAINTAINER {{.Maintainer}}
LABEL{{range $key, $value := .Labels}} {{$key}}="{{$value}}"{{end}}

# temp env hack until standard base image is available
ENV LANG='en_US.UTF-8' \
    TZ=Europe/Oslo \
    HOME=/u01

COPY ./app/application/bin $HOME/application/bin

RUN find $HOME/application -type d -exec chmod 755 {} + && \
	find $HOME/application -type f -exec chmod 644 {} + && \
	mkdir -p $HOME/logs && \
	chmod 777 $HOME/logs && \
	ln -s $HOME/logs $HOME/application/logs

ENV{{range $key, $value := .Env}} {{$key}}="{{$value}}"{{end}}

CMD ["/u01/bin/run.sh"]
`

type DockerfileData struct {
	BaseImage  string
	Maintainer string
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
	if meta.Python == nil {
		return errors.Errorf("Deliverable metadata does not contain \"Python\" element")
	} else if meta.Docker.Maintainer == "" {
		return errors.Errorf("Deliverable metadata does not contain \"Docker.Maintainer\" element")
	}

	return nil
}

func NewDockerFile(
	dockerSpec global.DockerSpec,
	auroraVersion runtime.AuroraVersion,
	meta config.DeliverableMetadata,
	baseImage runtime.DockerImage,
	imageBuildTime string) util.WriterFunc {
	return func(writer io.Writer) error {
		if err := verifyMetadata(meta); err != nil {
			return err
		}
		env := createEnv(auroraVersion, dockerSpec.PushExtraTags, imageBuildTime)

		data := &DockerfileData{
			BaseImage:  baseImage.GetCompleteDockerTagName(),
			Maintainer: meta.Docker.Maintainer,
			Labels:     createLabels(meta),
			Env:        env,
		}

		return util.NewTemplateWriter(data, "Dockerfile", dockerFileTemplateBody)(writer)
	}
}
