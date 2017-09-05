package prepare

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/nodejs/npm"
	"github.com/skatteetaten/architect/pkg/process/build"
	"github.com/skatteetaten/architect/pkg/util"
)

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

type NodeJsBuilder struct {
	Registry npm.Downloader
	provider docker.ImageInfoProvider
}

type PreparedImage struct {
	baseImage runtime.DockerImage
	Path      string
}

func Prepper(npmRegistry npm.Downloader) process.Prepper {
	return func(c *config.Config, provider docker.ImageInfoProvider) ([]docker.DockerBuildConfig, error) {
		builder := &NodeJsBuilder{
			Registry: npmRegistry,
			provider: provider,
		}

		preparedImages, err := builder.Prepare(c.NodeJsApplication, c.DockerSpec.GetExternalRegistryWithoutProtocol())
		if err != nil {
			return nil, err
		}

		buildConfigs := make([]docker.DockerBuildConfig, 0, 2)
		buildImage := runtime.ArchitectImage{
			Tag: c.BuilderSpec.Version,
		}
		for _, preparedImage := range preparedImages {
			auroraVersion := runtime.NewApplicationVersionFromBuilderAndBase(c.NodeJsApplication.Version, false,
				c.NodeJsApplication.Version, &buildImage, &preparedImage.baseImage)
			buildConfigs = append(buildConfigs, docker.DockerBuildConfig{
				BuildFolder:      preparedImage.Path,
				DockerRepository: c.DockerSpec.OutputRepository,
				AuroraVersion:    auroraVersion,
				Baseimage:        &preparedImage.baseImage,
			})
		}
		return buildConfigs, nil
	}
}

func (n *NodeJsBuilder) Prepare(c *config.NodeApplication, externalRegistry string) ([]PreparedImage, error) {
	logrus.Debug("Building %s", c.NpmName)
	applicaionName := c.NpmName
	version := c.Version

	packageJson, err := n.Registry.DownloadPackageJson(applicaionName)
	if err != nil {
		return nil, err
	}
	v, present := packageJson.Versions[version]
	if !present {
		return nil, errors.Errorf("No version %s for application %s", version, applicaionName)
	}
	if err != nil {
		return nil, err
	}

	nodejsBaseImageVersion, err := n.provider.GetCompleteBaseImageVersion(
		c.NodejsBaseImageSpec.BaseImage,
		c.NodejsBaseImageSpec.BaseVersion)

	if err != nil {
		return nil, err
	}
	logrus.Debugf("Nodejs base image version %s", nodejsBaseImageVersion)

	nodejsBaseImage := runtime.DockerImage{
		Repository: c.NodejsBaseImageSpec.BaseImage,
		Tag:        nodejsBaseImageVersion,
		Registry:   externalRegistry,
	}

	tarball, err := n.Registry.DownloadTarball(v.Dist.Tarball)

	if err != nil {
		return nil, err
	}
	// We must create separate folders for nodejs and nginx due to
	// Docker builds wanting one file per folder
	pathToApplication, err := npm.ExtractTarball(tarball)
	if err != nil {
		return nil, err
	}
	packageJsonFromPackage, err := npm.FindPackageJsonInsideTarball(tarball)
	if err != nil {
		return nil, err
	}
	imageBuildTime := docker.GetUtcTimestamp()
	err = prepareImage(packageJsonFromPackage, nodejsBaseImage, version, util.NewFileWriter(pathToApplication), imageBuildTime)
	if err != nil {
		return nil, err
	}
	logrus.Infof("Image build prepared in %s", pathToApplication)
	return []PreparedImage{{
		baseImage: nodejsBaseImage,
		Path:      pathToApplication,
	}}, nil
}

func prepareImage(v *npm.VersionedPackageJson, baseImage runtime.DockerImage, version string, writer util.FileWriter,
	imageBuildTime string) error {
	labels := make(map[string]string)
	labels["version"] = version
	labels["maintainer"] = findMaintainer(v.Maintainers)
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

func findMaintainer(maintainers []npm.Maintainer) string {
	if len(maintainers) == 0 {
		return "No Maintainer set!"
	}
	return maintainers[0].Name + " <" + maintainers[0].Email + ">"
}
