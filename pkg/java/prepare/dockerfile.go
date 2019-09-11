package prepare

import (
	"github.com/pkg/errors"
	global "github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/java/config"
	"github.com/skatteetaten/architect/pkg/util"
	"io"
)

// The base directory where all code is copied in the Docker image
const DockerBasedir = "/u01"

// Where in the build folder the application is put
const DockerfileApplicationFolder = "app"

const ApplicationFolder = "application"

// The directory where the application is prepared
const ApplicationBuildFolder = DockerfileApplicationFolder + "/" + ApplicationFolder

var legacyDockerFileTemplate string = `FROM {{.BaseImage}}

MAINTAINER {{.Maintainer}}
LABEL{{range $key, $value := .Labels}} {{$key}}="{{$value}}"{{end}}

COPY ./app $HOME
RUN chmod -R 777 $HOME && \
	ln -s $HOME/logs $HOME/application/logs && \
	rm $TRUST_STORE && \
	ln -s $HOME/architect/cacerts $TRUST_STORE

ENV{{range $key, $value := .Env}} {{$key}}="{{$value}}"{{end}}
`

var radishDockerFileTemplate string = `FROM {{.BaseImage}}

MAINTAINER {{.Maintainer}}
LABEL{{range $key, $value := .Labels}} {{$key}}="{{$value}}"{{end}}

COPY ./app radish.json $HOME/
RUN find $HOME/application -type d -exec chmod 755 {} + && \
	find $HOME/application -type f -exec chmod 644 {} + && \
	mkdir -p $HOME/logs && \
	chmod 777 $HOME/logs && \
	ln -s $HOME/logs $HOME/application/logs

ENV{{range $key, $value := .Env}} {{$key}}="{{$value}}"{{end}}
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

// TODO Consider moving this func
func verifyMetadata(meta config.DeliverableMetadata) error {
	if meta.Docker == nil {
		return errors.Errorf("Deliverable metadata does not contain \"Docker\" element")
	} else if meta.Docker.Maintainer == "" {
		return errors.Errorf("Deliverable metadata does not contain \"Docker.Maintainer\" element")
	}

	return nil
}

func NewRadishDockerFile(dockerSpec global.DockerSpec, auroraVersion runtime.AuroraVersion, meta config.DeliverableMetadata,
	baseImage runtime.DockerImage, imageBuildTime string) util.WriterFunc {
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

		return util.NewTemplateWriter(data, "Dockerfile", radishDockerFileTemplate)(writer)
	}
}

func NewLegacyDockerFile(dockerSpec global.DockerSpec, auroraVersion runtime.AuroraVersion, meta config.DeliverableMetadata,
	baseImage runtime.DockerImage, imageBuildTime string) util.WriterFunc {
	return func(writer io.Writer) error {

		if err := verifyMetadata(meta); err != nil {
			return err
		}
		env := createEnv(auroraVersion, dockerSpec.PushExtraTags, imageBuildTime)
		env["LOGBACK_FILE"] = "$HOME/architect/logback.xml"
		if meta.Openshift != nil {
			if meta.Openshift.ReadinessURL != "" {
				env[docker.ENV_READINESS_CHECK_URL] = meta.Openshift.ReadinessURL
			}

			if meta.Openshift.ReadinessOnManagementPort == "" || meta.Openshift.ReadinessOnManagementPort == "true" {
				env[docker.ENV_READINESS_ON_MANAGEMENT_PORT] = "true"
			}
		} else if meta.Java != nil && meta.Java.ReadinessURL != "" {
			env[docker.ENV_READINESS_CHECK_URL] = meta.Java.ReadinessURL
		}
		data := &DockerfileData{
			BaseImage:  baseImage.GetCompleteDockerTagName(),
			Maintainer: meta.Docker.Maintainer,
			Labels:     createLabels(meta),
			Env:        env,
		}

		return util.NewTemplateWriter(data, "Dockerfile", legacyDockerFileTemplate)(writer)
	}
}
