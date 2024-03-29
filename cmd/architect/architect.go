package architect

import (
	"context"
	"fmt"
	"github.com/skatteetaten/architect/v2/pkg/sporingslogger"
	"net/url"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/docker"
	doozer "github.com/skatteetaten/architect/v2/pkg/doozer/prepare"
	java "github.com/skatteetaten/architect/v2/pkg/java/prepare"
	"github.com/skatteetaten/architect/v2/pkg/nexus"
	nodejs "github.com/skatteetaten/architect/v2/pkg/nodejs/prepare"
	process "github.com/skatteetaten/architect/v2/pkg/process/build"
	"github.com/skatteetaten/architect/v2/pkg/process/retag"
)

var verbose bool

// RunConfiguration runtime configuration
type RunConfiguration struct {
	NexusDownloader         nexus.Downloader
	Config                  *config.Config
	RegistryCredentialsFunc func(string) (*docker.RegistryCredentials, error)
	PushUsername            string
	PushToken               string
}

// RunArchitect main
func RunArchitect(configuration RunConfiguration) {
	c := configuration.Config
	ctx := context.Background()
	startTimer := time.Now()
	logrus.Debugf("Config %+v", c)
	logrus.Infof("ARCHITECT_APP_VERSION=%s,ARCHITECT_AURORA_VERSION=%s", os.Getenv("APP_VERSION"), os.Getenv("AURORA_VERSION"))

	registryCredentials, err := configuration.getRegistryCredentials()
	if err != nil {
		logrus.Fatalf("Could not parse registry credentials %s", err)
	}

	logrus.Infof("Output registry %s", c.DockerSpec.OutputRegistry)

	pushRegistryURL := url.URL{
		Host:   c.DockerSpec.OutputRegistry,
		Scheme: "https",
	}

	pullRegistryURL, err := url.Parse(c.DockerSpec.InternalPullRegistry)
	if err != nil {
		logrus.Fatalf("Unable to parse URL %s", c.DockerSpec.InternalPullRegistry)
		return
	}

	pushRegistryConn := docker.RegistryConnectionInfo{
		Port:        docker.GetPortOrDefault(pushRegistryURL.Port()),
		Insecure:    docker.InsecureOrDefault(c),
		Host:        pushRegistryURL.Hostname(),
		Credentials: registryCredentials,
	}

	pullRegistryConn := docker.RegistryConnectionInfo{
		Port:        docker.GetPortOrDefault(pullRegistryURL.Port()),
		Host:        pullRegistryURL.Hostname(),
		Insecure:    docker.InsecureOrDefault(c),
		Credentials: nil,
	}
	pushRegistry := docker.NewRegistryClient(pushRegistryConn)
	pullRegistry := docker.NewRegistryClient(pullRegistryConn)

	sporingsLoggerClient := sporingslogger.NewClient(c.Sporingstjeneste)

	var builder process.Builder
	builder = process.NewLayerBuilder(c, pushRegistry, pullRegistry)

	if c.DockerSpec.RetagWith != "" {
		logrus.Info("Perform retag")
		if err := retag.Retag(ctx, c, registryCredentials, pullRegistry, builder); err != nil {
			logrus.Fatalf("Failed to retag temporary image %s", err)
		}
	} else {
		if err := performBuild(ctx, &configuration, c, pullRegistry, pushRegistry, builder, sporingsLoggerClient); err != nil {
			var errorMessage string
			if logrus.GetLevel() >= logrus.DebugLevel {
				errorMessage = "Failed to build image: %+v, Terminating"
			} else {
				errorMessage = "Failed to build image: %v, Terminating"
			}
			logrus.Fatal(fmt.Sprintf(errorMessage, err))
		}
	}

	logrus.Infof("Timer stage=RunArchitect apptype=%s registry=%s repository=%s timetaken=%.3fs", c.ApplicationType, c.DockerSpec.OutputRegistry, c.DockerSpec.OutputRepository, time.Since(startTimer).Seconds())
}
func performBuild(ctx context.Context, configuration *RunConfiguration, c *config.Config, pullRegistry docker.Registry,
	pushRegistry docker.Registry, builder process.Builder, sporingsLoggerClient sporingslogger.Sporingslogger) error {
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

	if c.BinaryBuild && c.BinaryBuildType == config.Snapshot && !c.ApplicationSpec.MavenGav.IsSnapshot() {
		logrus.Fatalf("Can only build snapshots. Make sure version ends with -SNAPSHOT")
	}

	ctx, cancel := context.WithTimeout(ctx, c.BuildTimeout*time.Second)
	defer cancel()

	return process.Build(ctx, pullRegistry, pushRegistry, c, configuration.NexusDownloader, prepper, builder, sporingsLoggerClient)
}

func (c RunConfiguration) getRegistryCredentials() (*docker.RegistryCredentials, error) {
	registry := c.Config.DockerSpec.OutputRegistry

	if c.PushToken != "" && c.PushUsername != "" {
		return &docker.RegistryCredentials{
			Username:      c.PushUsername,
			Password:      c.PushToken,
			Serveraddress: registry,
		}, nil
	}
	return c.RegistryCredentialsFunc(registry)

}
