package prepare

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/java/prepare/resources"
	"github.com/skatteetaten/architect/pkg/nexus"
	"github.com/skatteetaten/architect/pkg/process/build"
	"github.com/skatteetaten/architect/pkg/util"
	"strings"
)

type AuroraApplication struct {
	NodeJS            *NodeJSApplication `json:"nodejs"`
	Static            string             `json:"static"`
	ConfigurableProxy bool               `json:"configurableProxy"`
	Path              string             `json:"path"`
	SPA               bool               `json:"spa"`
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

COPY ./architectscripts /u01/architect

RUN chmod 755 /u01/architect/*

COPY ./{{.PackageDirectory}} /u01/application

COPY ./overrides /u01/bin/

COPY ./{{.PackageDirectory}}/{{.Static}} /u01/static{{.Path}}

COPY nginx.conf /etc/nginx/nginx.conf

RUN chmod 666 /etc/nginx/nginx.conf && \
    chmod 777 /etc/nginx && \
    chmod 755 /u01/bin/*

ENV MAIN_JAVASCRIPT_FILE="/u01/application/{{.MainFile}}" \
    IMAGE_BUILD_TIME="{{.ImageBuildTime}}" \
    PROXY_PASS_HOST="localhost" \
    PROXY_PASS_PORT="9090"

WORKDIR "/u01/"

CMD ["/u01/architect/run", "/u01/bin/run_nginx"]`

const START_SCRIPT string = `#!/bin/bash
source $HOME/architect/run_tools.sh
exec $1
`

//We copy this over the script in wrench if we don't have a nodejs app
const BLOCKING_RUN_NODEJS string = `#!/bin/sh
echo "No node. Blocking 4 ever<3!"
while true; do sleep 100d; done;
`

const READINESS_LIVENESS_SH = `#!/bin/sh
{{if .Include}}
wget --spider localhost:{{.Port}} > /dev/null 2>&1
{{end}}
`

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

       location /api {
          {{if or .HasNodeJSApplication .ConfigurableProxy}}proxy_pass http://${PROXY_PASS_HOST}:${PROXY_PASS_PORT};{{else}}return 404;{{end}}
       }
{{if .SPA}}
       location {{.Path}} {
          root /u01/static;
          try_files $uri {{.Path}}index.html;
       }
{{else}}
       location {{.Path}} {
          root /u01/static;
       }
{{end}}
    }
}
`

type PreparedImage struct {
	baseImage runtime.DockerImage
	Path      string
}

type probe struct {
	Include bool
	Port    int
}

type templateInput struct {
	Baseimage            string
	MainFile             string
	HasNodeJSApplication bool
	ConfigurableProxy    bool
	Static               string
	SPA                  bool
	Path                 string
	Labels               map[string]string
	PackageDirectory     string
	ImageBuildTime       string
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

	var path string
	if len(strings.TrimPrefix(v.Aurora.Path, "/")) == 0 {
		path = "/"
	} else {
		path = "/" + strings.TrimPrefix(v.Aurora.Path, "/")
	}

	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	var nodejsMainfile string
	if v.Aurora.NodeJS != nil {
		nodejsMainfile = strings.TrimSpace(v.Aurora.NodeJS.Main)
	}

	input := &templateInput{
		Baseimage:            baseImage.GetCompleteDockerTagName(),
		MainFile:             nodejsMainfile,
		HasNodeJSApplication: len(nodejsMainfile) != 0,
		ConfigurableProxy:    v.Aurora.ConfigurableProxy,
		Static:               v.Aurora.Static,
		SPA:                  v.Aurora.SPA,
		Path:                 path,
		Labels:               labels,
		PackageDirectory:     "package",
		ImageBuildTime:       imageBuildTime,
	}

	err := writer(util.NewTemplateWriter(input, "NgnixConfiguration", NGINX_CONFIG_TEMPLATE), "nginx.conf")
	if err != nil {
		return errors.Wrap(err, "Error creating nginx configuration")
	}
	err = writer(util.NewTemplateWriter(input, "NodejsDockerfile", WRENCH_DOCKER_FILE), "Dockerfile")
	if err != nil {
		return errors.Wrap(err, "Error creating Dockerfile")
	}
	//TODO: this peeks into Java package.. Should be refactored...
	bytes, err := resources.Asset("run_tools.sh")
	if err != nil {
		return errors.Wrap(err, "Could not find data")
	}
	err = writer(util.NewByteWriter(bytes), "architectscripts", "run_tools.sh")
	if err != nil {
		return errors.Wrap(err, "Failed add resource run_tools.sh")
	}
	err = writer(util.NewByteWriter([]byte(START_SCRIPT)), "architectscripts", "run")
	if err != nil {
		return errors.Wrap(err, "Failed creating run script")
	}
	err = addProbes(input, writer)
	if err != nil {
		return err
	}
	if !input.HasNodeJSApplication {
		err = writer(util.NewByteWriter([]byte(BLOCKING_RUN_NODEJS)), "overrides", "run_node")
		if err != nil {
			return errors.Wrap(err, "Failed creating nodejs override script")
		}
	}
	return err
}

func addProbes(input *templateInput, writer util.FileWriter) error {
	nginxProbe := &probe{
		Include: true,
		Port:    8080,
	}
	nodeProbe := &probe{
		Include: input.HasNodeJSApplication,
		Port:    9090,
	}
	err := writer(util.NewTemplateWriter(nodeProbe, "nodeliveness", READINESS_LIVENESS_SH),
		"overrides", "liveness_node.sh")
	if err != nil {
		return err
	}
	err = writer(util.NewTemplateWriter(nginxProbe, "nginxliveness", READINESS_LIVENESS_SH),
		"overrides", "liveness_nginx.sh")
	if err != nil {
		return err
	}
	err = writer(util.NewTemplateWriter(nodeProbe, "nodereadiness", READINESS_LIVENESS_SH),
		"overrides", "readiness_node.sh")
	if err != nil {
		return err
	}
	err = writer(util.NewTemplateWriter(nginxProbe, "nginxreadiness", READINESS_LIVENESS_SH),
		"overrides", "readiness_nginx.sh")
	if err != nil {
		return err
	}
	return nil
}

func findMaintainer(dockerMetadata DockerMetadata) string {
	if len(dockerMetadata.Maintainer) == 0 {
		return "No Maintainer set!"
	}
	return dockerMetadata.Maintainer
}
