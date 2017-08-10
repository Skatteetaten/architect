package java

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/java/nexus"
	"github.com/skatteetaten/architect/pkg/java/prepare"
	"github.com/skatteetaten/architect/pkg/process/build"
	"path"
	"strings"
)

func Prepper(downloader nexus.Downloader) process.Prepper {
	return func(cfg *config.Config, provider docker.ImageInfoProvider) ([]docker.DockerBuildConfig, error) {
		logrus.Debugf("Download deliverable for GAV %-v", cfg.JavaApplication)
		deliverable, err := downloader.DownloadArtifact(cfg.JavaApplication)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not download deliverable %-v", cfg.JavaApplication)
		}
		javaApplication := cfg.JavaApplication
		logrus.Debug("Extract build info")

		getAppVersionString := getAppVersion(cfg, deliverable.Path)
		completeBaseImageVersion, err := provider.GetCompleteBaseImageVersion(javaApplication.BaseImageSpec.BaseImage,
			javaApplication.BaseImageSpec.BaseVersion)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to get the complete build version")
		}

		baseImage := &runtime.BaseImage{
			Tag:        completeBaseImageVersion,
			Repository: javaApplication.BaseImageSpec.BaseImage,
			Registry:   cfg.DockerSpec.GetExternalRegistryWithoutProtocol(),
		}

		buildImage := &runtime.BuildImage{
			Tag: cfg.BuilderSpec.Version,
		}
		snapshot := strings.Contains(javaApplication.Version, "-SNAPSHOT")
		auroraVersion := runtime.NewApplicationVersionFromBuilderAndBase(getAppVersionString, snapshot,
			cfg.JavaApplication.Version, buildImage, baseImage)
		if err != nil {
			return nil, errors.Wrap(err, "Error creating version information")
		}

		logrus.Debug("Prepare output image")
		buildPath, err := prepare.Prepare(cfg.DockerSpec, auroraVersion, deliverable, baseImage)

		if err != nil {
			return nil, errors.Wrap(err, "Error prepare artifact")
		}

		buildConf := docker.DockerBuildConfig{
			AuroraVersion:    auroraVersion,
			BuildFolder:      buildPath,
			DockerRepository: cfg.DockerSpec.OutputRepository,
		}
		return []docker.DockerBuildConfig{buildConf}, nil
	}
}

/*
  Create app version. If not snapshot build, then return version from GAV.
  Otherwise, create new snapshot version based on deliverable.
*/
func getAppVersion(cfg *config.Config, deliverablePath string) string {
	if strings.Contains(cfg.JavaApplication.Version, "SNAPSHOT") {
		replacer := strings.NewReplacer(cfg.JavaApplication.ArtifactId, "", "-Leveransepakke.zip", "")
		version := "SNAPSHOT-" + replacer.Replace(path.Base(deliverablePath))
		return version
	}
	return cfg.JavaApplication.Version
}
