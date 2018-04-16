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
	"regexp"
	"strings"
)

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
          {{if or .HasNodeJSApplication .ConfigurableProxy}}proxy_pass http://${PROXY_PASS_HOST}:${PROXY_PASS_PORT};{{else}}return 404;{{end}}{{range $key, $value := .NginxOverrides}}
          {{$key}} {{$value}};{{end}}
       }
{{if .SPA}}
       location {{.Path}} {
          root /u01/static;
          try_files $uri {{.Path}}index.html;{{else}}
       location {{.Path}} {
          root /u01/static;{{end}}{{range $key, $value := .ExtraStaticHeaders}}
          add_header {{$key}} "{{$value}}";{{end}}
       }
    }
}
`

/*

We sanitize the input.... Don't want to large inputs.

For example; Accepting very large client_max_body_size would make a DOS attack very easy to implement...

*/
var allowedNginxOverrides = map[string]func(string) error{
	"client_max_body_size": func(s string) error {
		// between 1 and 20
		match, err := regexp.MatchString("^([1-9]|[1][0-9]|[2][0])m$", s)
		if err != nil {
			return err
		}
		if !match {
			return errors.New("Value on client_max_body_size should be on the form Nm where N is between 1 and 20")
		}
		return nil
	},
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

func prepareImage(v *openshiftJson, baseImage runtime.DockerImage, version string, writer util.FileWriter,
	imageBuildTime string) error {
	completeDockerName := baseImage.GetCompleteDockerTagName()
	input, err := mapOpenShiftJsonToTemplateInput(v, completeDockerName, imageBuildTime, version)

	if err != nil {
		return errors.Wrap(err, "Error processing AuroraConfig")
	}

	err = writer(util.NewTemplateWriter(input, "NgnixConfiguration", NGINX_CONFIG_TEMPLATE), "nginx.conf")
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

func findMaintainer(dockerMetadata dockerMetadata) string {
	if len(dockerMetadata.Maintainer) == 0 {
		return "No Maintainer set!"
	}
	return dockerMetadata.Maintainer
}

func mapOpenShiftJsonToTemplateInput(v *openshiftJson, completeDockerName string, imageBuildTime string, version string) (*templateInput, error) {
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
	var overrides map[string]string
	var err error
	if v.Aurora.NodeJS != nil {
		nodejsMainfile = strings.TrimSpace(v.Aurora.NodeJS.Main)
		overrides = v.Aurora.NodeJS.Overrides
		err = whitelistOverrides(overrides)
		if err != nil {
			return nil, err
		}
	}

	var static string
	var spa bool
	var extraHeaders map[string]string

	if v.Aurora.Webapp == nil {
		static = v.Aurora.Static
		spa = v.Aurora.SPA
		extraHeaders = nil

	} else {
		static = v.Aurora.Webapp.StaticContent
		spa = v.Aurora.Webapp.DisableTryfiles == false
		extraHeaders = v.Aurora.Webapp.Headers
	}

	return &templateInput{
		Baseimage:            completeDockerName,
		MainFile:             nodejsMainfile,
		HasNodeJSApplication: len(nodejsMainfile) != 0,
		NginxOverrides:       overrides,
		ConfigurableProxy:    v.Aurora.ConfigurableProxy,
		Static:               static,
		ExtraStaticHeaders:   extraHeaders,
		SPA:                  spa,
		Path:                 path,
		Labels:               labels,
		PackageDirectory:     "package",
		ImageBuildTime:       imageBuildTime,
	}, nil
}
func whitelistOverrides(overrides map[string]string) error {
	if overrides == nil {
		return nil
	}

	for key, value := range overrides {
		validateFunc, exists := allowedNginxOverrides[key]
		if !exists {
			return errors.New("Config " + key + " is not allowed to override with Architect.")
		}
		var err error
		if err = validateFunc(value); err != nil {
			return err
		}
	}
	return nil
}
