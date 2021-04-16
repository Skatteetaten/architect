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
	"net/url"
	"os"
	"strings"
	"time"
)

var verbose bool

//RunConfiguration runtime configuration
type RunConfiguration struct {
	NexusDownloader         nexus.Downloader
	Config                  *config.Config
	RegistryCredentialsFunc func(string) (*docker.RegistryCredentials, error)
}

//RunArchitect main
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

	pushRegistryUrl := url.URL{
		Host:   c.DockerSpec.OutputRegistry,
		Scheme: "https",
	}
	if err != nil {
		logrus.Fatalf("Unable to parse URL %s", c.DockerSpec.OutputRegistry)
	}
	pullRegistryUrl, err := url.Parse(c.DockerSpec.InternalPullRegistry)
	if err != nil {
		logrus.Fatalf("Unable to parse URL %s", c.DockerSpec.InternalPullRegistry)
		return
	}

	pushRegistryConn := docker.RegistryConnectionInfo{
		Port:        getPortOrDefault(pushRegistryUrl.Port()),
		Insecure:    insecureOrDefault(c),
		Host:        pushRegistryUrl.Hostname(),
		Credentials: registryCredentials,
	}

	pullRegistryConn := docker.RegistryConnectionInfo{
		Port:        getPortOrDefault(pullRegistryUrl.Port()),
		Host:        pullRegistryUrl.Hostname(),
		Insecure:    insecureOrDefault(c),
		Credentials: nil,
	}
	pushRegistry := docker.NewRegistryClient(pushRegistryConn)
	pullRegistry := docker.NewRegistryClient(pullRegistryConn)

	var builder process.Builder
	builder = process.NewLayerBuilder(c, pushRegistry, pullRegistry)

	if c.DockerSpec.RetagWith != "" {
		logrus.Info("Perform retag")
		//TODO: Vi må kanskje gjøre noe spesielt med flytting mellom snapshot og release?
		if err := retag.Retag(ctx, c, registryCredentials, pullRegistry, builder); err != nil {
			logrus.Fatalf("Failed to retag temporary image %s", err)
		}
	} else {
		err := performBuild(ctx, &configuration, c, pullRegistry, pushRegistry, builder)

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
func performBuild(ctx context.Context, configuration *RunConfiguration, c *config.Config, pullRegistry docker.Registry, pushRegistry docker.Registry, builder process.Builder) error {
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

	return process.Build(ctx, pullRegistry, pushRegistry, c, configuration.NexusDownloader, prepper, builder)
}

func getPortOrDefault(port string) string {
	if port == "" {
		return "443"
	}
	return port
}

//TODO: HACK: Fix registry certificate. TLS handshake fails with: does not contain any IP SANs
func insecureOrDefault(config *config.Config) bool {
	if config.BinaryBuild {
		return true
	}
	return false
}
