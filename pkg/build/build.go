package build

import (
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/cmd/architect"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/docker"
	"github.com/skatteetaten/architect/v2/pkg/nexus"
	"github.com/skatteetaten/architect/v2/pkg/util"
)

// Configuration binary build configuration
type Configuration struct {
	File             string
	ApplicationType  config.ApplicationType
	OutputRepository string
	TagWith          string
	BaseImageName    string
	BaseImageVersion string
	PushRegistry     string
	PullRegistry     string
	Version          string
	PushToken        string
	PushUsername     string
}

// BuildBinary run architect with binary input
func BuildBinary(c Configuration) {
	architectConfig := generateArchitectConfig(c)
	binaryInput, err := util.ExtractBinaryFromFile(c.File)
	if err != nil {
		logrus.Fatalf("Could not read binary input: %s", err)
	}

	nexusDownloader := nexus.NewBinaryDownloader(binaryInput)
	architect.RunArchitect(architect.RunConfiguration{
		NexusDownloader:         nexusDownloader,
		Config:                  architectConfig,
		RegistryCredentialsFunc: docker.LocalRegistryCredentials(),
		PushUsername:            c.PushUsername,
		PushToken:               c.PushToken,
	})
}

func generateArchitectConfig(c Configuration) *config.Config {
	return &config.Config{
		ApplicationType: c.ApplicationType,
		ApplicationSpec: config.ApplicationSpec{
			MavenGav: config.MavenGav{
				Version: c.Version,
			},
			BaseImageSpec: config.DockerBaseImageSpec{
				BaseImage:   c.BaseImageName,
				BaseVersion: c.BaseImageVersion,
			},
		},
		DockerSpec: config.DockerSpec{
			OutputRegistry:         c.PushRegistry,
			ExternalDockerRegistry: c.PushRegistry,
			InternalPullRegistry:   c.PullRegistry,
			OutputRepository:       c.OutputRepository,
			TagWith:                c.TagWith,
		},
		BinaryBuild:  true,
		LocalBuild:   true,
		BuildTimeout: 900,
	}
}
