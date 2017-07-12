package prepare

import (
	"github.com/pkg/errors"
	global "github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/java/config"
	"io"
	"text/template"
)

// The base directory where all code is copied in the Docker image
const DockerBasedir = "/u01"

// The directory where the application is prepared
const ApplicationDir = "app"

var dockerfileTemplate string = `FROM {{.BaseRepository}}:{{.BaseImageTag}}

MAINTAINER {{.Maintainer}}
LABEL {{range $key, $value := .Labels}}{{$key}}="{{$value}}" {{end}}

COPY ./app $HOME
RUN chmod -R 777 $HOME && \
	ln -s $HOME/logs $HOME/application/logs && \
	rm $TRUST_STORE && \
	ln -s $HOME/architect/cacerts $TRUST_STORE

ENV {{range $key, $value := .Env}}{{$key}}="{{$value}}" {{end}}
`

type Dockerfile struct {
	BaseRepository string
	BaseImageTag   string
	Maintainer     string
	Labels         map[string]string
	Env            map[string]string
}

func NewDockerfile(meta *config.DeliverableMetadata, buildinfo global.BuildInfo) (*Dockerfile, error) {
	if meta.Docker == nil {
		return nil, errors.Errorf("Deliverable metadata does not contain docker object")
	}

	// Maintainer
	maintainer := meta.Docker.Maintainer

	if maintainer == "" {
		return nil, errors.Errorf("Deliverable metadata does not contain maintainer element")
	}

	// Lables
	var labels map[string]string = make(map[string]string)

	for k, v := range meta.Docker.Labels {
		labels[k] = v
	}

	// Env
	var env map[string]string = make(map[string]string)

	for k, v := range buildinfo.Env {
		env[k] = v
	}

	appendArchitectEnv(env, meta)

	return &Dockerfile{buildinfo.BaseImage.Repository,
		buildinfo.BaseImage.Version, maintainer,
		labels, env}, nil
}

func (dockerfile *Dockerfile) Write(writer io.Writer) error {

	tmpl, err := template.New("dockerfile").Parse(dockerfileTemplate)

	if err != nil {
		return errors.Wrap(err, "Failed to parse Dockerfile template")
	}

	if err = tmpl.Execute(writer, dockerfile); err != nil {
		return errors.Wrap(err, "Failed to execute Dockerfile template")
	}

	return nil
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
