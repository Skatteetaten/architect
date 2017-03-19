package prepare

import (
	"text/template"
	"github.com/Skatteetaten/architect/pkg/java/config"
	"io"
)

type Dockerfile interface {
	Build(writer io.Writer) (error)
}

var dockerfileTemplate string =

	`FROM {{.DockerBase}}

	MAINTAINER {{.Maintainer}}
	LABEL {{range $key, $value := .Labels}}{{$key}}="{{$value}}" {{end}}

	COPY ./app /u01
	RUN chmod -R 777 /u01/

	ENV {{.Env}} \
	    {{range $key, $value := .ReadinessEnv}}{{$key}}="{{$value}}" {{end}}

	CMD ["bin/run"]`

type DefaultDockerfile struct {
	DockerBase   string
	Maintainer   string
	Labels       map[string]interface{}
	Env          string
	ReadinessEnv map[string]string
}

func NewForConfig(DockerBase string, Env string, cfg *config.ArchitectConfig) Dockerfile {
	var impl *DefaultDockerfile = &DefaultDockerfile{}
	impl.Maintainer = cfg.Docker.Maintainer
	impl.Labels = cfg.Docker.Labels.(map[string]interface{})
	impl.ReadinessEnv = findReadinessEnv(cfg)
	impl.Env = Env
	impl.DockerBase = DockerBase
	var spec Dockerfile = impl
	return spec
}

func (dockerfile *DefaultDockerfile) Build(writer io.Writer) (error) {

	tmpl, err := template.New("dockerfile").Parse(dockerfileTemplate)

	if err != nil {
		return err
	}

	return tmpl.Execute(writer, dockerfile)
}

func findReadinessEnv(cfg *config.ArchitectConfig) map[string]string {
	m := make(map[string]string)

	if cfg.Openshift != nil {
		if cfg.Openshift.ReadinessURL != "" {
			m["READINESS_CHECK_URL"] = cfg.Openshift.ReadinessURL
		}

		if cfg.Openshift.ReadinessOnManagementPort == "" || cfg.Openshift.ReadinessOnManagementPort == "true" {
			m["READINESS_ON_MANAGEMENT_PORT"] = "true"
		}
	} else if cfg.Java != nil && cfg.Java.ReadinessURL != "" {
		m["READINESS_CHECK_URL"] = cfg.Java.ReadinessURL
	}

	return m
}
