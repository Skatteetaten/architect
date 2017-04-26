package prepare

import (
	"github.com/skatteetaten/architect/pkg/java/config"
	global "github.com/skatteetaten/architect/pkg/config"
	"io"
	"text/template"
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

func NewDockerfile(meta *config.DeliverableMetadata, buildinfo global.BuildInfo, config global.Config) *Dockerfile {
	var maintainer string
	var labels map[string]string
	if meta.Docker != nil {
		maintainer = meta.Docker.Maintainer
		labels = meta.Docker.Labels
	}

	env := createEnv(buildinfo, config)

	appendReadinesEnv(env, meta)

	return &Dockerfile{buildinfo.BaseImage.Repository,
		buildinfo.BaseImage.Tags["INFERRED_VERSION"], maintainer,
		labels, env}
}

func (dockerfile *Dockerfile) Write(writer io.Writer) error {

	tmpl, err := template.New("dockerfile").Parse(dockerfileTemplate)

	if err != nil {
		return err
	}

	return tmpl.Execute(writer, dockerfile)
}

func createEnv(buildinfo global.BuildInfo, config global.Config) (map[string]string) {
	env := make(map[string]string)
	env["AURORA_VERSION"] = buildinfo.OutputImage.Tags["COMPLETE_VERSION"]
	env["APP_VERSION"] = config.MavenGav.Version

	return env
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
