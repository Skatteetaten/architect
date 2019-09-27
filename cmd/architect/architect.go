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

	if c.DockerSpec.RetagWith != "" {
		logrus.Info("Perform retag")
		if err := retag.Retag(c, registryCredentials); err != nil {
			logrus.Fatalf("Failed to retag temporary image %s", err)
		}
	} else {
		err := performBuild(&configuration, c, registryCredentials)

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
func performBuild(configuration *RunConfiguration, c *config.Config, r *docker.RegistryCredentials) error {
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

	provider := docker.NewRegistryClient(c.DockerSpec.ExternalDockerRegistry)

	if strings.Contains(strings.ToLower(c.BuildStrategy), "buildah") {
		logrus.Info("ALPHA FEATURE: Running buildah builds")
		buildah := &process.BuildahCmd{
			TlsVerify: c.TlsVerify,
		}
		return process.Build(r, provider, c, configuration.NexusDownloader, prepper, buildah)

	} else {
		if !strings.Contains(c.BuildStrategy, "docker") {
			logrus.Warnf("Unsupported build strategy: %s. Defaulting to docker", c.BuildStrategy)
		}

		dockerClient, err := process.NewDockerBuilder()
		if err != nil {
			return err
		}

		logrus.Info("Running docker build")
		return process.Build(r, provider, c, configuration.NexusDownloader, prepper, dockerClient)
	}
}
