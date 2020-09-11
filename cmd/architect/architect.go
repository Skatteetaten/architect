package architect

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/docker"
	doozer "github.com/skatteetaten/architect/pkg/doozer/prepare"
	java "github.com/skatteetaten/architect/pkg/java/prepare"
	"github.com/skatteetaten/architect/pkg/nexus"
	nodejs "github.com/skatteetaten/architect/pkg/nodejs/prepare"
	"github.com/skatteetaten/architect/pkg/process/build"
	"github.com/skatteetaten/architect/pkg/process/retag"
	"os"
	"strings"
	"time"
)

var verbose bool

type RunConfiguration struct {
	NexusDownloader         nexus.Downloader
	Config                  *config.Config
	RegistryCredentialsFunc func(string) (*docker.RegistryCredentials, error)
}

func RunArchitect(configuration RunConfiguration) {
	c := configuration.Config
	ctx := context.Background()
	startTimer := time.Now()
	logrus.Debugf("Config %+v", c)
	logrus.Infof("ARCHITECT_APP_VERSION=%s,ARCHITECT_AURORA_VERSION=%s", os.Getenv("APP_VERSION"), os.Getenv("AURORA_VERSION"))

	registryCredentials, err := configuration.RegistryCredentialsFunc(c.DockerSpec.OutputRegistry)

	if err != nil {
		logrus.Fatalf("Could not parse registry credentials %s", err)
	}

	provider := docker.NewRegistryClient(c.DockerSpec.InternalPullRegistry, c.DockerSpec.ExternalDockerRegistry, registryCredentials)

	var builder process.Builder
	builder = process.NewLayerBuilder(c, provider)

	if c.DockerSpec.RetagWith != "" {
		logrus.Info("Perform retag")
		if err := retag.Retag(ctx, c, registryCredentials, provider, builder); err != nil {
			logrus.Fatalf("Failed to retag temporary image %s", err)
		}
	} else {
		err := performBuild(ctx, &configuration, c, registryCredentials, provider, builder)

		if err != nil {
			var errorMessage string
			if logrus.GetLevel() >= logrus.DebugLevel {
				errorMessage = "Failed to build image: %+v, Terminating"
			} else {
				errorMessage = "Failed to build image: %v, Terminating"
			}

			errorMessage = fmt.Sprintf(errorMessage, err)

			if strings.Contains(errorMessage, "Cannot connect to the Docker daemon") {
				errorMessage = fmt.Sprintf("%s: The most likely cause is timeout", errorMessage)
			}

			logrus.Fatal(errorMessage)
		}
	}
	logrus.Infof("Timer stage=RunArchitect apptype=%s registry=%s repository=%s timetaken=%.3fs", c.ApplicationType, c.DockerSpec.OutputRegistry, c.DockerSpec.OutputRepository, time.Since(startTimer).Seconds())
}
func performBuild(ctx context.Context, configuration *RunConfiguration, c *config.Config, r *docker.RegistryCredentials, provider docker.Registry, builder process.Builder) error {
	var prepper process.Prepper
	if c.ApplicationType == config.JavaLeveransepakke {
		logrus.Info("Perform Java build")
		prepper = java.Prepper()
	} else if c.ApplicationType == config.NodeJsLeveransepakke {
		logrus.Info("Perform Webleveranse build")
		prepper = nodejs.Prepper()
	} else if c.ApplicationType == config.DoozerLeveranse {
		logrus.Info("Perform Doozerleveranse build")
		prepper = doozer.Prepper()
	}

	if !c.LocalBuild {
		if c.BinaryBuild && !c.ApplicationSpec.MavenGav.IsSnapshot() {
			logrus.Fatalf("Trying to build a release as binary build? Sorry, only SNAPSHOTS;)")
		}
	}

	ctx, cancel := context.WithTimeout(ctx, c.BuildTimeout*time.Second)
	defer cancel()

	return process.Build(ctx, r, provider, c, configuration.NexusDownloader, prepper, builder)
}
