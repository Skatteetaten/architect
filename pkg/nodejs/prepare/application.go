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

type ImageType int

const (
	NodeJSImage ImageType = iota
	NginxImage  ImageType = iota
)

const NODEJS_DOCKER_FILE string = `FROM {{.Baseimage}}

LABEL{{range $key, $value := .Labels}} {{$key}}="{{$value}}"{{end}}

COPY ./{{.PackageDirectory}} /u01/app

ENV MAIN_JAVASCRIPT_FILE="/u01/app/{{.MainFile}}"

WORKDIR "/u01/app"

CMD ["/u01/bin/run"]`

const NGINX_DOCKER_FILE string = `FROM {{.Baseimage}}

LABEL{{range $key, $value := .Labels}} {{$key}}="{{$value}}"{{end}}

COPY ./{{.PackageDirectory}}/{{.Static}} /u01/app/static

COPY nginx.conf /etc/nginx/nginx.conf
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
	Type      ImageType
	baseImage runtime.BaseImage
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
		nginxOutputRepository := c.DockerSpec.OutputRepository + "-static"
		nodejsOutputRepository := c.DockerSpec.OutputRepository + "-app"
		//TODO: AuroraVersion must be per image

		buildConfigs := make([]docker.DockerBuildConfig, 0, 2)
		buildImage := runtime.BuildImage{
			Tag: c.BuilderSpec.Version,
		}
		for _, preparedImage := range preparedImages {
			var outputRepository string
			if preparedImage.Type == NodeJSImage {
				outputRepository = nodejsOutputRepository
			} else {
				outputRepository = nginxOutputRepository
			}
			auroraVersion := runtime.NewApplicationVersionFromBuilderAndBase(c.NodeJsApplication.Version, false,
				c.NodeJsApplication.Version, &buildImage, &preparedImage.baseImage)
			buildConfigs = append(buildConfigs, docker.DockerBuildConfig{
				BuildFolder:      preparedImage.Path,
				DockerRepository: outputRepository,
				AuroraVersion:    auroraVersion,
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

	nginxBaseImageVersion, err := n.provider.GetCompleteBaseImageVersion(
		c.NginxBaseImageSpec.BaseImage,
		c.NginxBaseImageSpec.BaseVersion)

	if err != nil {
		return nil, err
	}
	logrus.Debugf("Nginx base image version %s", nginxBaseImageVersion)

	nodejsBaseImage := runtime.BaseImage{
		Repository: c.NodejsBaseImageSpec.BaseImage,
		Tag:        nodejsBaseImageVersion,
		Registry:   externalRegistry,
	}

	nginxBaseImage := runtime.BaseImage{
		Repository: c.NginxBaseImageSpec.BaseImage,
		Tag:        nginxBaseImageVersion,
		Registry:   externalRegistry,
	}

	tarball, err := n.Registry.DownloadTarball(v.Dist.Tarball)

	if err != nil {
		return nil, err
	}
	// We must create separate folders for nodejs and nginx due to
	// Docker builds wanting one file per folder
	pathToNodeJSApplication, err := npm.ExtractTarball(tarball)
	if err != nil {
		return nil, err
	}
	packageJsonFromPackage, err := npm.FindPackageJsonInsideTarball(tarball)
	if err != nil {
		return nil, err
	}
	err = prepareNodeJsImage(packageJsonFromPackage, nodejsBaseImage, version, util.NewFileWriter(pathToNodeJSApplication))
	if err != nil {
		return nil, err
	}
	logrus.Infof("NodeJS application build prepared in %s", pathToNodeJSApplication)
	pathToNginxApplication, err := npm.ExtractTarball(tarball)
	if err != nil {
		return nil, err
	}
	err = prepareNginxImage(packageJsonFromPackage, nginxBaseImage, version, util.NewFileWriter(pathToNginxApplication))
	if err != nil {
		return nil, err
	}
	logrus.Infof("Nginx build prepared in %s", pathToNginxApplication)
	return []PreparedImage{{
		Type:      NodeJSImage,
		baseImage: nodejsBaseImage,
		Path:      pathToNodeJSApplication,
	}, {
		Type:      NginxImage,
		baseImage: nginxBaseImage,
		Path:      pathToNginxApplication,
	}}, nil
}

func prepareNginxImage(v *npm.VersionedPackageJson, baseImage runtime.BaseImage, version string, writer util.FileWriter) error {
	labels := make(map[string]string)
	labels["version"] = version
	labels["maintainer"] = findMaintainer(v.Maintainers)
	input := &struct {
		Baseimage        string
		Static           string
		Labels           map[string]string
		PackageDirectory string
	}{
		Baseimage:        baseImage.GetDockerFileString(),
		Static:           v.Aurora.Static,
		Labels:           labels,
		PackageDirectory: "package",
	}
	err := writer(util.NewTemplateWriter(input, "NginxDockerfile", NGINX_DOCKER_FILE), "Dockerfile")
	if err != nil {
		return errors.Wrap(err, "Error creating dockerfile")
	}
	err = writer(util.NewTemplateWriter(input, "NgnixConfiguration", NGINX_CONFIG_TEMPLATE), "nginx.conf")
	if err != nil {
		return errors.Wrap(err, "Error creating nginx configuration")
	}
	return nil

}

func prepareNodeJsImage(v *npm.VersionedPackageJson, baseImage runtime.BaseImage, version string, writer util.FileWriter) error {
	labels := make(map[string]string)
	labels["version"] = version
	labels["maintainer"] = findMaintainer(v.Maintainers)
	input := &struct {
		Baseimage        string
		MainFile         string
		Labels           map[string]string
		PackageDirectory string
	}{
		Baseimage:        baseImage.GetDockerFileString(),
		MainFile:         v.Aurora.NodeJS.Main,
		Labels:           labels,
		PackageDirectory: "package",
	}
	f := util.NewTemplateWriter(input, "NodejsDockerfile", NODEJS_DOCKER_FILE)
	return writer(f, "Dockerfile")
}

func findMaintainer(maintainers []npm.Maintainer) string {
	if len(maintainers) == 0 {
		return "No Maintainer set!"
	}
	return maintainers[0].Name + " <" + maintainers[0].Email + ">"
}
