package prepare

import (
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/nexus"
	"github.com/skatteetaten/architect/pkg/util"
)

type BuildConfiguration struct {
	BuildContext string
	Env          map[string]string
	Labels       map[string]string
	Cmd          []string
}

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

func prepareLayers(dockerSpec config.DockerSpec, auroraVersion *runtime.AuroraVersion, deliverable nexus.Deliverable, baseImage runtime.BaseImage) (*BuildConfiguration, error) {
	openshiftJson, err := findOpenshiftJsonInTarball(deliverable.Path)
	if err != nil {
		return nil, err
	}

	buildPath, err := extractTarball(deliverable.Path)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Using build context: %s", buildPath)

	writer := util.NewFileWriter(buildPath)

	if err := os.MkdirAll(buildPath+"/layer/u01", 0755); err != nil {
		return nil, errors.Wrap(err, "Failed to create base layer structure")
	}

	if err := os.MkdirAll(buildPath+"/layer/u01/logs", 0777); err != nil {
		return nil, errors.Wrap(err, "Failed to create log folder")
	}

	imageBuildTime := docker.GetUtcTimestamp()
	completeDockerName := baseImage.GetCompleteDockerTagName()
	nginxData, dockerData, err := mapOpenShiftJsonToTemplateInput(dockerSpec, openshiftJson, completeDockerName, imageBuildTime, auroraVersion)
	if err != nil {
		return nil, errors.Wrap(err, "Failed while parsing openshift.json")
	}

	if err := writer(newRadishNginxConfig(dockerData, nginxData), "nginx-radish.json"); err != nil {
		return nil, errors.Wrap(err, "Unable to create radish-nginx configuration")
	}

	err = addProbes(nginxData.HasNodeJSApplication, writer)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to write health check probes")
	}

	if !nginxData.HasNodeJSApplication {
		err = writer(util.NewByteWriter([]byte(BLOCKING_RUN_NODEJS)), "overrides", "run_node")
		if err != nil {
			return nil, errors.Wrap(err, "Failed creating nodejs override script")
		}
	}

	//COPY ./{{.PackageDirectory}} /u01/application
	if err := os.MkdirAll(buildPath+"/layer/u01/application", 0755); err != nil {
		return nil, errors.Wrap(err, "Failed to create application folder")
	}

	err = util.CopyDirectory(buildPath+"/"+dockerData.PackageDirectory, buildPath+"/layer/u01/application")
	if err != nil {
		return nil, errors.Wrap(err, "Could not create application layer")
	}

	//COPY ./overrides /u01/bin/
	if err := os.MkdirAll(buildPath+"/layer/u01/bin", 0755); err != nil {
		return nil, errors.Wrap(err, "Failed to create bin folder")
	}
	err = util.CopyDirectory(buildPath+"/overrides", buildPath+"/layer/u01/bin")
	if err != nil {
		return nil, errors.Wrap(err, "Could not copy overrides")
	}

	//	COPY nginx-radish.json $HOME/
	err = util.Copy(buildPath+"/nginx-radish.json", buildPath+"/layer/u01/nginx-radish.json")
	if err != nil {
		return nil, errors.Wrap(err, "Could not copy nginx-radish.json")
	}
	//	COPY ./{{.PackageDirectory}}/{{.Static}} /u01/static{{.Path}}

	if err := os.MkdirAll(buildPath+"/layer/u01/static"+dockerData.Path, 0755); err != nil {
		return nil, errors.Wrap(err, "Failed to create static folder")
	}

	err = util.CopyDirectory(buildPath+"/"+dockerData.PackageDirectory+"/"+dockerData.Static, buildPath+"/layer/u01/static"+dockerData.Path)
	if err != nil {
		return nil, errors.Wrap(err, "Could not copy static files")
	}

	return &BuildConfiguration{
		BuildContext: buildPath,
		Env:          dockerData.Env,
		Labels:       dockerData.Labels,
		Cmd:          []string{"/u01/bin/run_nginx"},
	}, nil

}

func addProbes(hasNodejsApplication bool, writer util.FileWriter) error {
	nginxProbe := &probe{
		Include: true,
		Port:    8080,
	}
	nodeProbe := &probe{
		Include: hasNodejsApplication,
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

func mapOpenShiftJsonToTemplateInput(dockerSpec config.DockerSpec, v *openshiftJson, completeDockerName string, imageBuildTime string, auroraVersion *runtime.AuroraVersion) (*NginxfileData, *DockerfileData, error) {
	labels := make(map[string]string)
	if v.DockerMetadata.Labels != nil {
		for k, v := range v.DockerMetadata.Labels {
			labels[k] = v
		}
	}
	labels["version"] = string(auroraVersion.GetAppVersion())
	labels["maintainer"] = findMaintainer(v.DockerMetadata)

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
			return nil, nil, err
		}
	}

	var exclude []string
	var static string
	var spa bool
	var extraHeaders map[string]string
	var gZip = nginxGzip{}
	var nginxLocationMap = make(nginxLocations)
	if v.Aurora.Webapp == nil {
		static = v.Aurora.Static
		spa = v.Aurora.SPA
		exclude = nil
		extraHeaders = nil
		nginxLocationMap = nil
	} else {
		static = v.Aurora.Webapp.StaticContent
		spa = v.Aurora.Webapp.DisableTryfiles == false
		extraHeaders = v.Aurora.Webapp.Headers
		gZip = v.Aurora.Gzip
		nginxLocationMap = buildNginxLocations(v.Aurora.Locations)

		if v.Aurora.Exclude != nil {
			for _, value := range v.Aurora.Exclude {
				exclude = append(exclude, value)
			}
		}
	}

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

	return &NginxfileData{
			HasNodeJSApplication: len(nodejsMainfile) != 0,
			ConfigurableProxy:    v.Aurora.ConfigurableProxy,
			NginxOverrides:       overrides,
			Path:                 path,
			ExtraStaticHeaders:   extraHeaders,
			SPA:                  spa,
			Content:              static,
			Exclude:              exclude,
			Gzip:                 gZip,
			Locations:            nginxLocationMap,
		}, &DockerfileData{
			Main:             nodejsMainfile,
			Maintainer:       findMaintainer(v.DockerMetadata),
			Baseimage:        completeDockerName,
			PackageDirectory: "package",
			Static:           static,
			Labels:           labels,
			Env:              env,
			Path:             path,
		}, nil
}

func findMaintainer(dockerMetadata dockerMetadata) string {
	if len(dockerMetadata.Maintainer) == 0 {
		return "No Maintainer set!"
	}
	return dockerMetadata.Maintainer
}

func buildNginxLocations(locations map[string]interface{}) nginxLocations {
	if locations == nil || len(locations) == 0 {
		return nil
	}

	var nginxLocationMap = make(nginxLocations)

	for locKey, locValue := range locations {
		myMap := locValue.(map[string]interface{})
		gZip := nginxGzip{}

		if val, ok := myMap["gzip"]; ok {
			gzipMap := val.(map[string]interface{})

			if val, ok := gzipMap["use_static"]; ok {
				gZip.UseStatic = val.(string)
			}
		}

		headersMap := make(map[string]string)
		if val, ok := myMap["headers"]; ok {
			for headerKey, headerValue := range val.(map[string]interface{}) {
				headersMap[headerKey] = headerValue.(string)
			}
		}
		nginxLocationMap[locKey] = &nginxLocation{headersMap, gZip}
	}
	return nginxLocationMap
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
