package prepare

import (
	"github.com/skatteetaten/architect/pkg/java/config"
	global "github.com/skatteetaten/architect/pkg/config"
	"io"
	"text/template"
	"github.com/pkg/errors"
)

const ENV_APP_VERSION = "APP_VERSION"
const ENV_AURORA_VERSION = "AURORA_VERSION"
const ENV_SNAPSHOT_TAG = "SNAPSHOT_TAG"

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

func NewDockerfile(meta *config.DeliverableMetadata, buildinfo global.BuildInfo) *Dockerfile {
	var maintainer string
	var labels map[string]string
	if meta.Docker != nil {
		maintainer = meta.Docker.Maintainer
		labels = meta.Docker.Labels
	}

	env := initEnv(buildinfo)

	appendReadinesEnv(env, meta)

	return &Dockerfile{buildinfo.BaseImage.Repository,
		buildinfo.BaseImage.Tags["INFERRED"], maintainer,
		labels, env}
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

func initEnv(buildinfo global.BuildInfo) (map[string]string) {
	env := make(map[string]string)
	env[ENV_AURORA_VERSION] = buildinfo.AuroraVersion
	env[ENV_APP_VERSION] = buildinfo.AppVersion

	if buildinfo.SnapshotVersion != "" {
		env[ENV_SNAPSHOT_TAG] = buildinfo.SnapshotVersion
	}

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
