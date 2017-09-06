package process

import (
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/nexus"
)

// Prepper is a fuction used to prepare a docker image. It is called within the context of
// The
type Prepper func(
	cfg *config.Config,
	auroraVersion *runtime.AuroraVersion,
	deliverable nexus.Deliverable,
	baseImage runtime.DockerImage) ([]docker.DockerBuildConfig, error)
