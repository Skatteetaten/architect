package doozer

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/doozer/prepare"
	"github.com/skatteetaten/architect/pkg/nexus"
	"github.com/skatteetaten/architect/pkg/process/build"
	"strings"
)

func Prepper() process.Prepper {
	return func(cfg *config.Config, auroraVersion *runtime.AuroraVersion, deliverable nexus.Deliverable,
		baseImage runtime.BaseImage) ([]docker.DockerBuildConfig, error) {

		if strings.ToLower(cfg.BuildStrategy) == config.Layer {
			return nil, errors.New("Doozer layer build not supported")
		}

		logrus.Debug("Pull output image")
		buildPath, err := prepare.Prepare(cfg.DockerSpec, auroraVersion, deliverable, baseImage)

		if err != nil {
			return nil, errors.Wrap(err, "Error prepare artifact")
		}

		buildConf := docker.DockerBuildConfig{
			AuroraVersion:    auroraVersion,
			BuildFolder:      buildPath,
			DockerRepository: cfg.DockerSpec.OutputRepository,
			Baseimage:        baseImage.DockerImage,
		}
		return []docker.DockerBuildConfig{buildConf}, nil
	}
}
