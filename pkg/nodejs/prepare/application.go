package prepare

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/nexus"
	"github.com/skatteetaten/architect/pkg/process/build"
	"github.com/skatteetaten/architect/pkg/util"
)

type AuroraApplication struct {
	NodeJS NodeJSApplication `json:"nodejs"`
	Static string            `json:"static"`
}

type NodeJSApplication struct {
	Main    string `json:"main"`
	Waf     string `json:"waf"`
	Runtime string `json:"runtime"`
}

type OpenshiftJson struct {
	Aurora         AuroraApplication `json:"web"`
	DockerMetadata DockerMetadata    `json:"docker"`
}

type DockerMetadata struct {
	Maintainer string            `json:"name"`
	Labels     map[string]string `json:"labels"`
}

const WRENCH_DOCKER_FILE string = `FROM {{.Baseimage}}

LABEL{{range $key, $value := .Labels}} {{$key}}="{{$value}}"{{end}}

COPY ./{{.PackageDirectory}} /u01/app

COPY ./{{.PackageDirectory}}/{{.Static}} /u01/app/static

COPY nginx.conf /etc/nginx/nginx.conf

ENV MAIN_JAVASCRIPT_FILE="/u01/app/{{.MainFile}}" IMAGE_BUILD_TIME="{{.ImageBuildTime}}"

WORKDIR "/u01/app"

CMD ["/u01/bin/run_node"]`

const NGINX_CONFIG_TEMPLATE string = `
worker_processes  1;
error_log stderr;

events {
    worker_connections  1024;
}


http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';

    access_log  /dev/stdout;

    sendfile        on;
    #tcp_nopush     on;

    keepalive_timeout  65;

    #gzip  on;

    index index.html;

    server {
       listen 8080;
       root /u01/app/static;

       location /api {
          proxy_pass http://localhost:9090;
       }

    }
}
`

type PreparedImage struct {
	baseImage runtime.DockerImage
	Path      string
}

func Prepper() process.Prepper {
	return func(cfg *config.Config, auroraVersion *runtime.AuroraVersion, deliverable nexus.Deliverable,
		baseImage runtime.DockerImage) ([]docker.DockerBuildConfig, error) {

		preparedImages, err := prepare(cfg.ApplicationSpec, auroraVersion, deliverable, baseImage)
		if err != nil {
			return nil, err
		}

		buildConfigs := make([]docker.DockerBuildConfig, 0, 2)
		for _, preparedImage := range preparedImages {
			buildConfigs = append(buildConfigs, docker.DockerBuildConfig{
				BuildFolder:      preparedImage.Path,
				DockerRepository: cfg.DockerSpec.OutputRepository,
				AuroraVersion:    auroraVersion,
				Baseimage:        preparedImage.baseImage,
			})
		}
		return buildConfigs, nil
	}
}

func prepare(c config.ApplicationSpec, auroraVersion *runtime.AuroraVersion,
	deliverable nexus.Deliverable, baseImage runtime.DockerImage) ([]PreparedImage, error) {
	logrus.Debug("Building %s", c.MavenGav.Name())

	openshiftJson, err := findOpenshiftJsonInTarball(deliverable.Path)
	if err != nil {
		return nil, err
	}

	pathToApplication, err := extractTarball(deliverable.Path)
	if err != nil {
		return nil, err
	}

	imageBuildTime := docker.GetUtcTimestamp()
	err = prepareImage(openshiftJson, baseImage, string(auroraVersion.GetAppVersion()), util.NewFileWriter(pathToApplication), imageBuildTime)
	if err != nil {
		return nil, err
	}
	logrus.Infof("Image build prepared in %s", pathToApplication)
	return []PreparedImage{{
		baseImage: baseImage,
		Path:      pathToApplication,
	}}, nil
}

func prepareImage(v *OpenshiftJson, baseImage runtime.DockerImage, version string, writer util.FileWriter,
	imageBuildTime string) error {
	labels := make(map[string]string)
	if v.DockerMetadata.Labels != nil {
		for k, v := range v.DockerMetadata.Labels {
			labels[k] = v
		}
	}
	labels["version"] = version
	labels["maintainer"] = findMaintainer(v.DockerMetadata)

	input := &struct {
		Baseimage        string
		MainFile         string
		Static           string
		Labels           map[string]string
		PackageDirectory string
		ImageBuildTime   string
	}{
		Baseimage:        baseImage.GetCompleteDockerTagName(),
		MainFile:         v.Aurora.NodeJS.Main,
		Static:           v.Aurora.Static,
		Labels:           labels,
		PackageDirectory: "package",
		ImageBuildTime:   imageBuildTime,
	}
	err := writer(util.NewTemplateWriter(input, "NgnixConfiguration", NGINX_CONFIG_TEMPLATE), "nginx.conf")
	if err != nil {
		return errors.Wrap(err, "Error creating nginx configuration")
	}
	f := util.NewTemplateWriter(input, "NodejsDockerfile", WRENCH_DOCKER_FILE)
	return writer(f, "Dockerfile")
}

func findMaintainer(dockerMetadata DockerMetadata) string {
	if len(dockerMetadata.Maintainer) == 0 {
		return "No Maintainer set!"
	}
	return dockerMetadata.Maintainer
}
