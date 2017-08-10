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

// The root of the application
const ApplicationRoot = "app"

// The directory where the application is prepared
const ApplicationFolder = ApplicationRoot + "/application"

var dockerfileTemplate string = `FROM {{.BaseRepository}}:{{.BaseImageTag}}

MAINTAINER {{.Maintainer}}
LABEL{{range $key, $value := .Labels}} {{$key}}="{{$value}}"{{end}}

COPY ./app $HOME
RUN chmod -R 777 $HOME && \
	ln -s $HOME/logs $HOME/application/logs && \
	rm $TRUST_STORE && \
	ln -s $HOME/architect/cacerts $TRUST_STORE

ENV{{range $key, $value := .Env}} {{$key}}="{{$value}}"{{end}}
`

type Dockerfile struct {
	BaseRepository string
	BaseImageTag   string
	Maintainer     string
	Labels         map[string]string
	Env            map[string]string
}

func NewDockerfile(dockerSpec global.DockerSpec, auroraVersion *runtime.AuroraVersion, meta *config.DeliverableMetadata, baseImage *runtime.BaseImage) util.WriterFunc {
	return func(writer io.Writer) error {

		env := createEnv(auroraVersion, dockerSpec.PushExtraTags)

		if meta.Docker == nil {
			return errors.Errorf("Deliverable metadata does not contain docker object")
		}

		// Maintainer
		maintainer := meta.Docker.Maintainer

		if maintainer == "" {
			return errors.Errorf("Deliverable metadata does not contain maintainer element")
		}

		// Lables
		var labels map[string]string = make(map[string]string)

		for k, v := range meta.Docker.Labels {
			labels[k] = v
		}

		appendArchitectEnv(env, meta)

		dockerFile := &Dockerfile{
			BaseRepository: baseImage.Repository,
			BaseImageTag:   baseImage.Tag,
			Maintainer:     maintainer,
			Labels:         labels,
			Env:            env}
		return util.NewTemplateWriter(dockerFile, "Dockerfile", dockerfileTemplate)(writer)
	}
}

func appendArchitectEnv(env map[string]string, meta *config.DeliverableMetadata) {

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

	env["LOGBACK_FILE"] = "$HOME/architect/logback.xml"
}
