package process

import (
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/docker"
)

// Prepper is a fuction used to prepare a docker image. It is called within the context of
// The
type Prepper func(config *config.Config, registry docker.ImageInfoProvider) ([]docker.DockerBuildConfig, error)
