package prepare

import (
	"fmt"
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
	"sort"
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

ENV{{range $key, $value := .Env}} {{$key}}="{{$value}}"{{end}}

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

{{.Gzip}}

    index index.html;

    server {
        listen 8080;

        location /api {
            {{if or .HasNodeJSApplication .ConfigurableProxy}}proxy_pass http://${PROXY_PASS_HOST}:${PROXY_PASS_PORT};{{else}}return 404;{{end}}{{range $key, $value := .NginxOverrides}}
            {{$key}} {{$value}};{{end}}
        }
{{if .SPA}}
        location {{.Path}} {
            root {{.DocumentRoot}};
            try_files $uri {{.Path}}index.html;{{else}}
        location {{.Path}} {
            root {{.DocumentRoot}};{{end}}{{range $key, $value := .ExtraStaticHeaders}}
            add_header {{$key}} "{{$value}}";{{end}}
        }
{{.Locations}}
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
		match, err := regexp.MatchString("^([1-9]|[1-4][0-9]|[5][0])m$", s)
		if err != nil {
			return err
		}
		if !match {
			return errors.New("Value on client_max_body_size should be on the form Nm where N is between 1 and 50")
		}
		return nil
	},
}

func Prepper() process.Prepper {
	return func(cfg *config.Config, auroraVersion *runtime.AuroraVersion, deliverable nexus.Deliverable,
		baseImage runtime.BaseImage) ([]docker.DockerBuildConfig, error) {

		preparedImages, err := prepare(*cfg, auroraVersion, deliverable, baseImage.DockerImage)
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

/*
func prepare(dockerSpec config.DockerSpec, c config.ApplicationSpec, auroraVersion *runtime.AuroraVersion,
	deliverable nexus.Deliverable, baseImage runtime.DockerImage) ([]PreparedImage, error) {*/
func prepare(cfg config.Config, auroraVersion *runtime.AuroraVersion,
	deliverable nexus.Deliverable, baseImage runtime.DockerImage) ([]PreparedImage, error) {
	logrus.Debug("Building %s", cfg.ApplicationSpec.MavenGav.Name())

	openshiftJson, err := findOpenshiftJsonInTarball(deliverable.Path)
	if err != nil {
		return nil, err
	}

	pathToApplication, err := extractTarball(deliverable.Path)
	if err != nil {
		return nil, err
	}

	imageBuildTime := docker.GetUtcTimestamp()
	err = prepareImage(cfg.DockerSpec, openshiftJson, baseImage, auroraVersion, util.NewFileWriter(pathToApplication), imageBuildTime)
	if err != nil {
		return nil, err
	}
	logrus.Infof("Image build prepared in %s", pathToApplication)
	return []PreparedImage{{
		baseImage: baseImage,
		Path:      pathToApplication,
	}}, nil
}

func prepareImage(dockerSpec config.DockerSpec, v *openshiftJson, baseImage runtime.DockerImage, auroraVersion *runtime.AuroraVersion, writer util.FileWriter,
	imageBuildTime string) error {
	completeDockerName := baseImage.GetCompleteDockerTagName()
	input, err := mapOpenShiftJsonToTemplateInput(dockerSpec, v, completeDockerName, imageBuildTime, auroraVersion)

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

func mapOpenShiftJsonToTemplateInput(dockerSpec config.DockerSpec, v *openshiftJson, completeDockerName string, imageBuildTime string, auroraVersion *runtime.AuroraVersion) (*templateInput, error) {
	labels := make(map[string]string)
	if v.DockerMetadata.Labels != nil {
		for k, v := range v.DockerMetadata.Labels {
			labels[k] = v
		}
	}
	labels["version"] = string(auroraVersion.GetAppVersion())
	labels["maintainer"] = findMaintainer(v.DockerMetadata)

	documentRoot := "/u01/static"
	path := "/"
	if v.Aurora.Webapp != nil && len(strings.TrimPrefix(v.Aurora.Webapp.Path, "/")) > 0 {
		path = "/" + strings.TrimPrefix(v.Aurora.Webapp.Path, "/")
	} else if len(strings.TrimPrefix(v.Aurora.Path, "/")) > 0 {
		logrus.Warnf("web.path in openshift.json is deprecated. Please use web.webapp.path when setting path: %s", v.Aurora.Path)
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
	nginxLocationMap := make(nginxLocations)

	gzipUse := "on"
	gzipMinLength := 10240
	gzipVary := "on"
	if v.Aurora.Webapp != nil && v.Aurora.Webapp.Gzip != nil {
		if v.Aurora.Webapp.Gzip.Use == "on" || v.Aurora.Webapp.Gzip.Use == "off" {
			gzipUse = v.Aurora.Webapp.Gzip.Use
			if v.Aurora.Webapp.Gzip.Use == "on" || v.Aurora.Webapp.Gzip.Use == "off" {
				gzipVary = v.Aurora.Webapp.Gzip.Use
			}
			if v.Aurora.Webapp.Gzip.MinLength > 0 {
				gzipMinLength = v.Aurora.Webapp.Gzip.MinLength
			}
		}
	}
	nginxRootGzip := nginxGzip{Use: gzipUse, MinLength: gzipMinLength, Vary: gzipVary}
	locationTryFiles := ""

	if v.Aurora.Webapp == nil {
		static = v.Aurora.Static
		spa = v.Aurora.SPA
		extraHeaders = nil
	} else {
		static = v.Aurora.Webapp.StaticContent
		spa = v.Aurora.Webapp.DisableTryfiles == false
		extraHeaders = v.Aurora.Webapp.Headers

		//  Locations
		for locKey, locValue := range v.Aurora.Webapp.Locations {
			myMap := locValue.(map[string]interface{})
			gZip := nginxGzip{}

			if val, ok := myMap["gzip"]; ok {
				gzipMap := val.(map[string]interface{})
				if val, ok := gzipMap["use"]; ok {
					gZip.Use = val.(string)
				}

				if val, ok := gzipMap["min_length"]; ok {
					gZip.MinLength = int(val.(float64))
				}

				if val, ok := gzipMap["vary"]; ok {
					gZip.Vary = val.(string)
				}
			}
			headersMap := make(map[string]string)

			for headerKey, headerValue := range myMap["headers"].(map[string]interface{}) {
				headersMap[headerKey] = headerValue.(string)
			}
			nginxLocationMap[locKey] = nginxLocation{headersMap, gZip}
		}
	}

	nginxGzipForTemplate := nginxGzipMapToString(nginxRootGzip)

	if spa {
		locationTryFiles = path + "index.html"
	}

	nginxLocationForTemplate := nginxLocationsMapToString(nginxLocationMap, documentRoot, path, locationTryFiles)

	env := make(map[string]string)
	env["MAIN_JAVASCRIPT_FILE"] = "/u01/application/" + nodejsMainfile
	env["PROXY_PASS_HOST"] = "localhost"
	env["PROXY_PASS_PORT"] = "9090"
	env[docker.IMAGE_BUILD_TIME] = imageBuildTime
	env[docker.ENV_APP_VERSION] = string(auroraVersion.GetAppVersion())
	env[docker.ENV_AURORA_VERSION] = string(auroraVersion.GetCompleteVersion())
	env[docker.ENV_PUSH_EXTRA_TAGS] = dockerSpec.PushExtraTags.ToStringValue()
	if auroraVersion.Snapshot {
		env[docker.ENV_SNAPSHOT_TAG] = auroraVersion.GetGivenVersion()
	}

	return &templateInput{
		Baseimage:            completeDockerName,
		HasNodeJSApplication: len(nodejsMainfile) != 0,
		NginxOverrides:       overrides,
		ConfigurableProxy:    v.Aurora.ConfigurableProxy,
		Static:               static,
		DocumentRoot:         documentRoot,
		ExtraStaticHeaders:   extraHeaders,
		SPA:                  spa,
		Path:                 path,
		Labels:               labels,
		Env:                  env,
		Locations:            nginxLocationForTemplate,
		Gzip:                 nginxGzipForTemplate,
		PackageDirectory:     "package",
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

func (m nginxLocations) sort() (index []string) {
	for k := range m {
		index = append(index, k)
	}
	sort.Strings(index)
	return
}

func (m headers) sort() (index []string) {
	for k := range m {
		index = append(index, k)
	}
	sort.Strings(index)
	return
}

func nginxLocationsMapToString(m nginxLocations, documentRoot string, path string, tryFiles string) string {
	sumLocations := ""
	indentN1 := strings.Repeat(" ", 8)
	indentN2 := strings.Repeat(" ", 12)

	for _, k := range m.sort() {
		v := m[k]
		singleLocation := fmt.Sprintf("%slocation %s%s {\n", indentN1, path, k)
		singleLocation = fmt.Sprintf("%s%sroot %s;\n", singleLocation, indentN2, documentRoot)
		gZipUse := strings.TrimSpace(v.Gzip.Use)
		gZipVary := strings.TrimSpace(v.Gzip.Vary)
		if gZipUse == "on" {
			singleLocation = fmt.Sprintf("%s%sgzip on;\n", singleLocation, indentN2)
			if v.Gzip.MinLength > 0 {
				singleLocation = fmt.Sprintf("%s%sgzip_min_length %d;\n", singleLocation, indentN2, v.Gzip.MinLength)
			}
			if gZipVary != "" {
				singleLocation = fmt.Sprintf("%s%sgzip_vary %s;\n", singleLocation, indentN2, gZipVary)
			}
			if v.Gzip.Proxied != "" {
				singleLocation = fmt.Sprintf("%s%sgzip_proxied %s;\n", singleLocation, indentN2, v.Gzip.Proxied)
			}
			if v.Gzip.Types != "" {
				singleLocation = fmt.Sprintf("%s%sgzip_types %s;\n", singleLocation, indentN2, v.Gzip.Types)
			}
			if v.Gzip.Disable != "" {
				singleLocation = fmt.Sprintf("%s%sgzip_disable \"%s\";\n", singleLocation, indentN2, v.Gzip.Disable)
			}
		} else if gZipUse == "off" {
			singleLocation = fmt.Sprintf("%s%sgzip off;\n", singleLocation, indentN2)
		}

		for _, k2 := range v.Headers.sort() {
			singleLocation = fmt.Sprintf("%s%sadd_header %s \"%s\";\n", singleLocation, indentN2, k2, v.Headers[k2])
		}

		if tryFiles != "" {
			singleLocation = fmt.Sprintf("%s%stry_files $uri %s;\n", singleLocation, indentN2, tryFiles)
		}

		singleLocation = fmt.Sprintf("%s%s}\n", singleLocation, indentN1)
		sumLocations = sumLocations + singleLocation
	}
	return sumLocations
}

func nginxGzipMapToString(gzip nginxGzip) string {
	sumGzip := ""
	indent := strings.Repeat(" ", 4)
	if gzip.Use == "on" {
		sumGzip = fmt.Sprintf("%s%sgzip on;\n", sumGzip, indent)
		if gzip.MinLength > 0 {
			sumGzip = fmt.Sprintf("%s%sgzip_min_length %d;\n", sumGzip, indent, gzip.MinLength)
		}
		if gzip.Vary != "" {
			sumGzip = fmt.Sprintf("%s%sgzip_vary %s;\n", sumGzip, indent, gzip.Vary)
		}
		if gzip.Proxied != "" {
			sumGzip = fmt.Sprintf("%s%sgzip_proxied %s;\n", sumGzip, indent, gzip.Proxied)
		}
		if gzip.Types != "" {
			sumGzip = fmt.Sprintf("%s%sgzip_types %s;\n", sumGzip, indent, gzip.Types)
		}
		if gzip.Disable != "" {
			sumGzip = fmt.Sprintf("%s%sgzip_disable \"%s\";\n", sumGzip, indent, gzip.Disable)
		}
	} else {
		sumGzip = fmt.Sprintf("%s%sgzip off;\n", sumGzip, indent)
	}
	return sumGzip
}
