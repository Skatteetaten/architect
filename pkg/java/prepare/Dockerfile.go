package prepare

import (
	"text/template"
	"../config"
	"io"
)

type Dockerfile interface {
	Build(DockerBase string, Env string, config config.ArchitectConfig, writer io.Writer) (error)
}

type DockerParam struct {
	DockerBase string
	Maintainer string
	Labels string
	Env string
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
}

func CreateDefaultImpl() *Dockerfile {
	var impl *DefaultDockerfile = &DefaultDockerfile{}
	var dockerfile Dockerfile = impl
	return &dockerfile
}

func (dockerfile *DefaultDockerfile) Build(DockerBase string, Env string, config config.ArchitectConfig, writer io.Writer) (error) {

	tmpl, err := template.New("dockerfile").Parse(dockerfileTemplate)

	if err != nil {
		return err
	}

	params := DockerParam{DockerBase, config.GetMaintainer(), config.GetLabels(), Env, config.GetReadinessEnv()}

	err = tmpl.Execute(writer, params)

	if err != nil {
		return err
	}

	return nil
}


