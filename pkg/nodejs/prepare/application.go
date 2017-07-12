package prepare

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
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

const NODEJS_DOCKER_FILE string = `FROM aurora/wrench:latest

LABEL{{range $key, $value := .Labels}} {{$key}}="{{$value}}"{{end}}

COPY ./{{.PackageDirectory}} /u01/app

ENV MAIN_JAVASCRIPT_FILE="/u01/app/{{.MainFile}}"

WORKDIR "/u01/app"

CMD ["/u01/bin/run"]`

const NGINX_DOCKER_FILE string = `FROM aurora/modem:latest

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
}

type PreparedImage struct {
	Type ImageType
	Path string
}

func Prepper(npmRegistry npm.Downloader) process.Prepper {
	return func(c *config.Config, registry docker.ImageInfoProvider) ([]docker.DockerBuildConfig, error) {
		builder := &NodeJsBuilder{
			Registry: npmRegistry,
		}
		preparedImages, err := builder.Prepare(c)
		if err != nil {
			return nil, err
		}
		auroraVersions, err := config.NewAuroraVersions(c.NodeJSGav.Version, false, c.NodeJSGav.Version,
			c.DockerSpec, c.BuilderSpec, "BASEIMAGEVERSION!!")
		if err != nil {
			return nil, err
		}
		buildsConfigs := make([]docker.DockerBuildConfig, len(preparedImages))
		tags, err := auroraVersions.GetAppVersion().GetVersionTags(c.DockerSpec.PushExtraTags)
		if err != nil {
			return nil, err
		}
		outputRegistry := c.DockerSpec.OutputRegistry
		nginxOutputRepository := c.DockerSpec.OutputRepository + "-static"
		nodejsOutputRepository := c.DockerSpec.OutputRepository + "-app"
		for i, preparedImage := range preparedImages {
			var outputRepository string
			if preparedImage.Type == NodeJSImage {
				outputRepository = nodejsOutputRepository
			} else {
				outputRepository = nginxOutputRepository
			}
			buildsConfigs[i] = docker.DockerBuildConfig{
				BuildFolder: preparedImage.Path,
				Tags:        docker.CreateImageNameFromSpecAndTags(tags, outputRegistry, outputRepository),
			}
		}
		return buildsConfigs, nil
	}
}

func (n *NodeJsBuilder) Prepare(c *config.Config) ([]PreparedImage, error) {
	logrus.Debug("Building %s", c.NodeJSGav.NpmName)
	applicaionName := c.NodeJSGav.NpmName
	version := c.NodeJSGav.Version
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
	err = prepareNodeJsImage(&v, version, util.NewFileWriter(pathToNodeJSApplication))
	if err != nil {
		return nil, err
	}
	logrus.Infof("NodeJS application build prepared in %s", pathToNodeJSApplication)
	pathToNginxApplication, err := npm.ExtractTarball(tarball)
	if err != nil {
		return nil, err
	}
	err = prepareNginxImage(&v, version, util.NewFileWriter(pathToNginxApplication))
	if err != nil {
		return nil, err
	}
	logrus.Infof("Nginx build prepared in %s", pathToNginxApplication)
	return []PreparedImage{{
		Type: NodeJSImage,
		Path: pathToNodeJSApplication,
	}, {
		Type: NginxImage,
		Path: pathToNginxApplication,
	}}, nil
}

func prepareNginxImage(v *npm.Version, version string, writer util.FileWriter) error {
	labels := make(map[string]string)
	labels["version"] = version
	labels["maintainer"] = findMaintainer(v.Maintainers)
	input := &struct {
		Static           string
		Labels           map[string]string
		PackageDirectory string
	}{
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

func prepareNodeJsImage(v *npm.Version, version string, writer util.FileWriter) error {
	labels := make(map[string]string)
	labels["version"] = version
	labels["maintainer"] = findMaintainer(v.Maintainers)
	input := &struct {
		MainFile         string
		Labels           map[string]string
		PackageDirectory string
	}{
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
