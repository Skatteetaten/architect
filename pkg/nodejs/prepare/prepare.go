package prepare

import (
	process "github.com/skatteetaten/architect/v2/pkg/process/build"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"github.com/skatteetaten/architect/v2/pkg/docker"
	"github.com/skatteetaten/architect/v2/pkg/nexus"
	"github.com/skatteetaten/architect/v2/pkg/util"
)

type BuildConfiguration struct {
	BuildContext string
	Env          map[string]string
	Labels       map[string]string
	Cmd          []string
}

func Prepper() process.Prepper {
	return func(cfg *config.Config, auroraVersion *runtime.AuroraVersion, deliverable nexus.Deliverable,
		baseImage runtime.BaseImage) (*docker.BuildConfig, error) {

		buildConfiguration, err := prepareLayers(cfg.DockerSpec, auroraVersion, deliverable, baseImage)
		if err != nil {
			return nil, errors.Wrap(err, "Error while preparing layers")
		}

		return &docker.BuildConfig{
			AuroraVersion:    auroraVersion,
			DockerRepository: cfg.DockerSpec.OutputRepository,
			BuildFolder:      buildConfiguration.BuildContext,
			Image:            baseImage.DockerImage,
			Env:              buildConfiguration.Env,
			Labels:           buildConfiguration.Labels,
			Cmd:              buildConfiguration.Cmd,
		}, nil
	}
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

	if err := util.MkdirAllWithPermissions(buildPath+"/layer/u01", 0755); err != nil {
		return nil, errors.Wrap(err, "Failed to create base layer structure")
	}

	if err := util.MkdirAllWithPermissions(buildPath+"/layer/u01/logs", 0777); err != nil {
		return nil, errors.Wrap(err, "Failed to create log folder")
	}

	imageBuildTime := docker.GetUtcTimestamp()
	completeDockerName := baseImage.GetCompleteDockerTagName()
	nginxData, dockerData, err := mapOpenShiftJSONToTemplateInput(dockerSpec, openshiftJson, completeDockerName, imageBuildTime, auroraVersion)
	if err != nil {
		return nil, errors.Wrap(err, "Failed while parsing openshift.json")
	}

	if err := writer(newRadishNginxConfig(dockerData, nginxData), "nginx-radish.json"); err != nil {
		return nil, errors.Wrap(err, "Unable to create radish-nginx configuration")
	}

	if err := addNginxProbes(buildPath, writer); err != nil {
		return nil, errors.Wrap(err, "Nginx: Unable to write health check probes")
	}

	if err := addNodeProbes(buildPath, nginxData.HasNodeJSApplication, writer); err != nil {
		return nil, errors.Wrap(err, "Node: Unable to write health check probes")
	}

	if !nginxData.HasNodeJSApplication {
		err = writer(util.NewByteWriter([]byte(BlockingRunNodeJS)), "overrides", "run_node")
		if err != nil {
			return nil, errors.Wrap(err, "Failed creating nodejs override script")
		}
		err = os.Chmod(buildPath+"/overrides/run_node", 0755)
		if err != nil {
			return nil, errors.Wrap(err, "Could not set file permissions")
		}
	}

	//COPY ./{{.PackageDirectory}} /u01/application
	if err := util.MkdirAllWithPermissions(buildPath+"/layer/u01/application", 0755); err != nil {
		return nil, errors.Wrap(err, "Failed to create application folder")
	}

	err = util.CopyDirectory(buildPath+"/"+dockerData.PackageDirectory, buildPath+"/layer/u01/application")
	if err != nil {
		return nil, errors.Wrap(err, "Could not create application layer")
	}

	//COPY ./overrides /u01/bin/
	if err := util.MkdirAllWithPermissions(buildPath+"/layer/u01/bin", 0755); err != nil {
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

	if err := util.MkdirAllWithPermissions(buildPath+"/layer/u01/static"+dockerData.Path, 0755); err != nil {
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

func addNginxProbes(buildPath string, writer util.FileWriter) error {

	nginxProbe := &probe{
		Include: true,
		Port:    8080,
	}
	err := writer(util.NewTemplateWriter(nginxProbe, "nginxreadiness", ReadinessLivenessSH),
		"overrides", "readiness_nginx.sh")
	if err != nil {
		return err
	}
	err = os.Chmod(buildPath+"/overrides/readiness_nginx.sh", 0755)
	if err != nil {
		return errors.Wrap(err, "Could not set file permissions")
	}

	err = writer(util.NewTemplateWriter(nginxProbe, "nginxliveness", ReadinessLivenessSH),
		"overrides", "liveness_nginx.sh")
	if err != nil {
		return err
	}
	err = os.Chmod(buildPath+"/overrides/liveness_nginx.sh", 0755)
	if err != nil {
		return errors.Wrap(err, "Could not set file permissions")
	}

	return nil
}

func addNodeProbes(buildPath string, hasNodejsApplication bool, writer util.FileWriter) error {
	nodeProbe := &probe{
		Include: hasNodejsApplication,
		Port:    9090,
	}
	err := writer(util.NewTemplateWriter(nodeProbe, "nodeliveness", ReadinessLivenessSH),
		"overrides", "liveness_node.sh")
	if err != nil {
		return err
	}
	err = os.Chmod(buildPath+"/overrides/liveness_node.sh", 0755)
	if err != nil {
		return errors.Wrap(err, "Could not set file permissions")
	}

	err = writer(util.NewTemplateWriter(nodeProbe, "nodereadiness", ReadinessLivenessSH),
		"overrides", "readiness_node.sh")
	if err != nil {
		return err
	}
	err = os.Chmod(buildPath+"/overrides/readiness_node.sh", 0755)
	if err != nil {
		return errors.Wrap(err, "Could not set file permissions")
	}
	return nil
}

func mapOpenShiftJSONToTemplateInput(dockerSpec config.DockerSpec, v *openshiftJSON, completeDockerName string, imageBuildTime string, auroraVersion *runtime.AuroraVersion) (*NginxfileData, *ImageMetadata, error) {
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
	if v.Aurora.NodeJS != nil {
		nodejsMainfile = strings.TrimSpace(v.Aurora.NodeJS.Main)
		overrides = v.Aurora.NodeJS.Overrides
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
	env[docker.ImageBuildTime] = imageBuildTime
	env[docker.EnvAppVersion] = string(auroraVersion.GetAppVersion())
	env[docker.EnvAuroraVersion] = string(auroraVersion.GetCompleteVersion())
	env[docker.EnvPushExtraTags] = dockerSpec.PushExtraTags.ToStringValue()
	if auroraVersion.Snapshot {
		env[docker.EnvSnapshotVersion] = auroraVersion.GetGivenVersion()
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
		}, &ImageMetadata{
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
