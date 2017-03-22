package prepare

import (
	"github.com/skatteetaten/architect/pkg/java/config"
	"io"
	"text/template"
)

type Dockerfile interface {
	Build(writer io.Writer) error
}

var dockerfileTemplate string = `FROM {{.DockerBase}}

MAINTAINER {{.Maintainer}}
LABEL {{range $key, $value := .Labels}}{{$key}}="{{$value}}" {{end}}

COPY ./app /u01
RUN chmod -R 777 /u01/

ENV {{range $key, $value := .Env}}{{$key}}="{{$value}}" {{end}}

CMD ["bin/run"]`

type TemplateDockerfile struct {
	DockerBase string
	Maintainer string
	Labels     map[string]string
	Env        map[string]string
}

func NewTemplateDockerfile(dockerBase string, env map[string]string, meta *config.DeliverableMetadata) Dockerfile {
	var maintainer string
	var labels map[string]string
	if meta.Docker != nil {
		maintainer = meta.Docker.Maintainer
		labels = meta.Docker.Labels
	}

	appendReadinesEnv(env, meta)

	return &TemplateDockerfile{dockerBase, maintainer, labels, env}
}

func (dockerfile *TemplateDockerfile) Build(writer io.Writer) error {

	tmpl, err := template.New("dockerfile").Parse(dockerfileTemplate)

	if err != nil {
		return err
	}

	return tmpl.Execute(writer, dockerfile)
}

func appendReadinesEnv(env map[string]string, meta *config.DeliverableMetadata) {

	if meta.Openshift != nil {
		if meta.Openshift.ReadinessURL != "" {
			env["READINESS_CHECK_URL"] = meta.Openshift.ReadinessURL
		}

		if meta.Openshift.ReadinessOnManagementPort == "" || meta.Openshift.ReadinessOnManagementPort == "true" {
			env["READINESS_ON_MANAGEMENT_PORT"] = "true"
		}
	} else if meta.Java != nil && meta.Java.ReadinessURL != "" {
		env["READINESS_CHECK_URL"] = meta.Java.ReadinessURL
	}
}
