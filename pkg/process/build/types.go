package process

import (
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"github.com/skatteetaten/architect/v2/pkg/docker"
	"github.com/skatteetaten/architect/v2/pkg/nexus"
)

// Prepper is a fuction used to prepare a docker image. It is called within the context of
// the prepare stage
type Prepper func(
	cfg *config.Config,
	auroraVersion *runtime.AuroraVersion,
	deliverable nexus.Deliverable,
	baseImage runtime.BaseImage) (*docker.BuildConfig, error)
