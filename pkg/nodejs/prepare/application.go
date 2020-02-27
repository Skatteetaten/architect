package prepare

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/nexus"
	process "github.com/skatteetaten/architect/pkg/process/build"
	"github.com/skatteetaten/architect/pkg/util"
)

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

		preparedImages, err := prepare(*cfg, auroraVersion, deliverable, baseImage)
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
	deliverable nexus.Deliverable, baseImage runtime.BaseImage) ([]PreparedImage, error) {
	logrus.Debugf("Building %s", cfg.ApplicationSpec.MavenGav.Name())

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
		baseImage: baseImage.DockerImage,
		Path:      pathToApplication,
	}}, nil
}

func prepareImage(dockerSpec config.DockerSpec, v *openshiftJson, baseImage runtime.BaseImage, auroraVersion *runtime.AuroraVersion, writer util.FileWriter,
	imageBuildTime string) error {
	completeDockerName := baseImage.GetCompleteDockerTagName()
	nginxData, dockerData, err := mapOpenShiftJsonToTemplateInput(dockerSpec, v, completeDockerName, imageBuildTime, auroraVersion)

	if err != nil {
		return errors.Wrap(err, "Error processing AuroraConfig")
	}

	if architecture, exists := baseImage.ImageInfo.Labels["www.skatteetaten.no-imageArchitecture"]; exists && architecture == "nodejs" {

		logrus.Info("Running radish nodejs build")

		if err := writer(newRadishNginxConfig(dockerData, nginxData), "nginx-radish.json"); err != nil {
			return errors.Wrap(err, "Unable to create radish-nginx configuration")
		}

		err = writer(util.NewTemplateWriter(dockerData, "NodejsDockerfile", WRENCH_RADISH_DOCKER_FILE), "Dockerfile")
		if err != nil {
			return errors.Wrap(err, "Error creating Dockerfile")
		}

	} else {
		logrus.Info("Running nodejs legacy build...")

		err = writer(util.NewTemplateWriter(nginxData, "NgnixConfiguration", NGINX_CONFIG_TEMPLATE), "nginx.conf")
		if err != nil {
			return errors.Wrap(err, "Error creating nginxData configuration")
		}
		err = writer(util.NewTemplateWriter(dockerData, "NodejsDockerfile", WRENCH_DOCKER_FILE), "Dockerfile")
		if err != nil {
			return errors.Wrap(err, "Error creating Dockerfile")
		}
	}
	err = addProbes(nginxData.HasNodeJSApplication, writer)
	if err != nil {
		return err
	}
	if !nginxData.HasNodeJSApplication {
		err = writer(util.NewByteWriter([]byte(BLOCKING_RUN_NODEJS)), "overrides", "run_node")
		if err != nil {
			return errors.Wrap(err, "Failed creating nodejs override script")
		}
	}
	return err
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

func findMaintainer(dockerMetadata dockerMetadata) string {
	if len(dockerMetadata.Maintainer) == 0 {
		return "No Maintainer set!"
	}
	return dockerMetadata.Maintainer
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
				if val, ok := gzipMap["vary"]; ok {
					gZip.Vary = val.(string)
				}

				if val, ok := gzipMap["proxied"]; ok {
					gZip.Proxied = val.(string)
				}

				if val, ok := gzipMap["disable"]; ok {
					gZip.Disable = val.(string)
				}

				if val, ok := gzipMap["http_version"]; ok {
					gZip.HttpVersion = val.(string)
				}
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
