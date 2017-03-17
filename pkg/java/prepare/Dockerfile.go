package prepare

import (
	"text/template"
	"../config"
	"io"
	"fmt"
	"strings"
)

type Dockerfile interface {
	Build(DockerBase string, Env string, writer io.Writer) (error)
}

type DockerParam struct {
	DockerBase   string
	Maintainer   string
	Labels       string
	Env          string
	ReadinessEnv string
}

var dockerfileTemplate string =

	`FROM {{.DockerBase}}

	MAINTAINER {{.Maintainer}}
	LABEL {{.Labels}}

	COPY ./app /u01
	RUN chmod -R 777 /u01/

	ENV {{.Env}} \
	    {{.ReadinessEnv}}

	CMD ["bin/run"]`

type DefaultDockerfile struct {
	maintainer   string
	labels       string
	readinessEnv string
}

func NewForConfig(cfg *config.ArchitectConfig) Dockerfile {
	var impl *DefaultDockerfile = &DefaultDockerfile{}
	impl.maintainer = findMaintainer(cfg)
	impl.labels = findLabels(cfg)
	impl.readinessEnv = findReadinessEnv(cfg)
	var spec Dockerfile = impl
	return spec
}

func (dockerfile *DefaultDockerfile) Build(DockerBase string, Env string, writer io.Writer) (error) {

	tmpl, err := template.New("dockerfile").Parse(dockerfileTemplate)

	if err != nil {
		return err
	}

	params := DockerParam{DockerBase, dockerfile.maintainer, dockerfile.labels, Env, dockerfile.readinessEnv}

	err = tmpl.Execute(writer, params)

	if err != nil {
		return err
	}

	return nil
}

func findMaintainer(cfg *config.ArchitectConfig) string {
	return cfg.Docker.Maintainer
}

func findLabels(cfg *config.ArchitectConfig) string {
	if cfg.Docker.Labels == nil {
		return ""
	} else {
		return joinMap(cfg.Docker.Labels.(map[string]interface{}))
	}
}

func findReadinessEnv(cfg *config.ArchitectConfig) string {
	m := make(map[string]interface{})

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

	return joinMap(m)
}

func joinMap(m map[string]interface{}) string {
	var labels []string

	for k, v := range m {
		labels = append(labels, fmt.Sprintf("%s=\"%s\"", k, v))
	}

	return strings.Join(labels, " ")
}
