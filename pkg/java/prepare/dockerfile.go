package prepare

import (
	"github.com/skatteetaten/architect/pkg/java/config"
	global "github.com/skatteetaten/architect/pkg/config"
	"io"
	"text/template"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/docker"
)

var dockerfileTemplate string = `FROM {{.BaseRepository}}:{{.BaseImageTag}}

MAINTAINER {{.Maintainer}}
LABEL {{range $key, $value := .Labels}}{{$key}}="{{$value}}" {{end}}

COPY ./app /u01
RUN chmod -R 777 /u01/

ENV {{range $key, $value := .Env}}{{$key}}="{{$value}}" {{end}}

CMD ["bin/run"]`

type Dockerfile struct {
	BaseRepository 	string
	BaseImageTag	string
	Maintainer 	string
	Labels     	map[string]string
	Env        	map[string]string
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

	appendReadinesEnv(env, meta)

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

func appendReadinesEnv(env map[string]string, meta *config.DeliverableMetadata) {

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
}
