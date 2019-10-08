package architect

import (
	"github.com/Sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/java"
	"github.com/skatteetaten/architect/pkg/nexus"
	"github.com/skatteetaten/architect/pkg/nodejs/prepare"
	"github.com/skatteetaten/architect/pkg/process/build"
	"github.com/skatteetaten/architect/pkg/process/retag"
	"os"
	"strings"
	"time"
)

var localRepo bool
var verbose bool

type RunConfiguration struct {
	NexusDownloader         nexus.Downloader
	Config                  *config.Config
	RegistryCredentialsFunc func(string) (*docker.RegistryCredentials, error)
}

func RunArchitect(configuration RunConfiguration) {
	c := configuration.Config
	startTimer := time.Now()
	logrus.Debugf("Config %+v", c)
	logrus.Infof("ARCHITECT_APP_VERSION=%s,ARCHITECT_AURORA_VERSION=%s", os.Getenv("APP_VERSION"), os.Getenv("AURORA_VERSION"))

	registryCredentials, err := configuration.RegistryCredentialsFunc(c.DockerSpec.OutputRegistry)

	if err != nil {
		logrus.Fatalf("Could not parse registry credentials %s", err)
	}



	var builder process.Builder

	if strings.Contains(strings.ToLower(c.BuildStrategy), config.Buildah) {
		logrus.Info("ALPHA FEATURE: Running buildah builds")
		builder = &process.BuildahCmd{
			TlsVerify: c.TlsVerify,
		}

	} else {
		if !strings.Contains(c.BuildStrategy, config.Docker) {
			logrus.Warnf("Unsupported build strategy: %s. Defaulting to docker", c.BuildStrategy)
		}

		builder, err = process.NewDockerBuilder()
		if err != nil {
			logrus.Fatal("err", err)
		}

		logrus.Info("Running docker build")
	}

	if c.DockerSpec.RetagWith != "" {
		provider := docker.NewRegistryClient(c.DockerSpec.InternalPullRegistry)
		logrus.Info("Perform retag")
		if err := retag.Retag(c, registryCredentials, provider, builder); err != nil {
			logrus.Fatalf("Failed to retag temporary image %s", err)
		}
	} else {
		provider := docker.NewRegistryClient(c.DockerSpec.InternalPullRegistry)
		err := performBuild(&configuration, c, registryCredentials, provider, builder)

		if err != nil {
			var errorMessage string
			if logrus.GetLevel() >= logrus.DebugLevel {
				errorMessage = "Failed to build image: %+v, Terminating"
			} else {
				errorMessage = "Failed to build image: %v, Terminating"
			}
			logrus.Fatalf(errorMessage, err)
		}
	}
	logrus.Infof("Timer stage=RunArchitect apptype=%s registry=%s repository=%s timetaken=%.3fs", c.ApplicationType, c.DockerSpec.OutputRegistry, c.DockerSpec.OutputRepository, time.Since(startTimer).Seconds())
}
func performBuild(configuration *RunConfiguration, c *config.Config, r *docker.RegistryCredentials, provider docker.ImageInfoProvider, builder process.Builder) error {
	var prepper process.Prepper
	if c.ApplicationType == config.JavaLeveransepakke {
		logrus.Info("Perform Java build")
		prepper = java.Prepper()
	} else if c.ApplicationType == config.NodeJsLeveransepakke {
		logrus.Info("Perform Webleveranse build")
		prepper = prepare.Prepper()
	}

	if c.BinaryBuild && !c.ApplicationSpec.MavenGav.IsSnapshot() {
		logrus.Fatalf("Trying to build a release as binary build? Sorry, only SNAPSHOTS;)")
	}

	return process.Build(r, provider, c, configuration.NexusDownloader, prepper, builder)

}
