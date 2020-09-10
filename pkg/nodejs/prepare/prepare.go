package prepare

import (
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/nexus"
	process "github.com/skatteetaten/architect/pkg/process/build"
)

func Prepper() process.Prepper {
	return func(cfg *config.Config, auroraVersion *runtime.AuroraVersion, deliverable nexus.Deliverable,
		baseImage runtime.BaseImage) ([]docker.DockerBuildConfig, error) {

		buildConfiguration, err := prepareLayers(cfg.DockerSpec, auroraVersion, deliverable, baseImage)
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
	}
}
