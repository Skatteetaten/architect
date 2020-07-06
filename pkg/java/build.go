package java

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/java/prepare"
	"github.com/skatteetaten/architect/pkg/nexus"
	"github.com/skatteetaten/architect/pkg/process/build"
	"strings"
)

func Prepper() process.Prepper {
	return func(cfg *config.Config, auroraVersion *runtime.AuroraVersion, deliverable nexus.Deliverable,
		baseImage runtime.BaseImage) ([]docker.DockerBuildConfig, error) {

		logrus.Debug("Pull output image")

		if strings.ToLower(cfg.BuildStrategy) == config.Layer {
			buildConfiguration, err := prepare.PrepareLayers(cfg.DockerSpec, auroraVersion, deliverable, baseImage)
			if err != nil {
				return nil, errors.Wrap(err, "Error while preparing layers")
			}
			return []docker.DockerBuildConfig{{
				AuroraVersion:    auroraVersion,
				DockerRepository: cfg.DockerSpec.OutputRepository,
				BuildFolder:      buildConfiguration.BuildContext,
				Baseimage:        baseImage.DockerImage,
				Env:              buildConfiguration.Env,
				Labels:           buildConfiguration.Labels,
				Cmd:              buildConfiguration.Cmd,
			}}, nil

		} else {
			buildPath, err := prepare.Prepare(cfg.DockerSpec, auroraVersion, deliverable, baseImage)
			if err != nil {
				return nil, errors.Wrap(err, "Error while preparing the docker context")
			}
			return []docker.DockerBuildConfig{{
				AuroraVersion:    auroraVersion,
				BuildFolder:      buildPath,
				DockerRepository: cfg.DockerSpec.OutputRepository,
				Baseimage:        baseImage.DockerImage,
			}}, nil
		}
	}
}
