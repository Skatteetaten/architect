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
	"github.com/skatteetaten/architect/pkg/util"
	"github.com/spf13/cobra"
	"os"
	"time"
)

var localRepo bool
var verbose bool

type RunConfiguration struct {
	NexusDownloader         nexus.Downloader
	Config                  *config.Config
	RegistryCredentialsFunc func(string) (*docker.RegistryCredentials, error)
}

var JavaLeveransepakke = &cobra.Command{

	Use:   "build",
	Short: "Build Docker image from Zip",
	Long:  `TODO`,
	Run: func(cmd *cobra.Command, args []string) {
		var configReader = config.NewInClusterConfigReader()
		var nexusDownloader nexus.Downloader
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		} else {
			logrus.SetLevel(logrus.InfoLevel)
		}
		if len(cmd.Flag("fileconfig").Value.String()) != 0 {
			conf := cmd.Flag("fileconfig").Value.String()
			logrus.Debugf("Using config from %s", conf)
			configReader = config.NewFileConfigReader(conf)
		}

		// Read build config
		c, err := configReader.ReadConfig()
		if err != nil {
			logrus.Fatalf("Could not read configuration: %s", err)
		}

		var binaryInput string
		if c.BinaryBuild {
			binaryInput, err = util.ExtractBinaryFromStdIn()
			if err != nil {
				logrus.Fatalf("Could not read binary input: %s", err)
			}
		}

		if c.BinaryBuild {
			nexusDownloader = nexus.NewBinaryDownloader(binaryInput)
		} else {
			mavenRepo := "http://aurora/nexus/service/local/artifact/maven/content"
			logrus.Debugf("Using Maven repo on %s", mavenRepo)
			nexusDownloader = nexus.NewNexusDownloader(mavenRepo)
		}

		RunArchitect(RunConfiguration{
			NexusDownloader:         nexusDownloader,
			Config:                  c,
			RegistryCredentialsFunc: docker.LocalRegistryCredentials(),
		})
	},
}

func init() {
	JavaLeveransepakke.Flags().StringP("fileconfig", "f", "", "Path to file config. If not set, the environment variable BUILD is read")
	JavaLeveransepakke.Flags().StringP("skippush", "s", "", "If set, Docker push will not be performed")
	JavaLeveransepakke.Flags().BoolVarP(&localRepo, "binary", "b", false, "If set, the Leveransepakke will be fetched from stdin")
	JavaLeveransepakke.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
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
		performBuild(&configuration, c, registryCredentials)

	}
	logrus.Infof("Timer stage=RunArchitect apptype=%s registry=%s repository=%s timetaken=%.3fs", c.ApplicationType, c.DockerSpec.OutputRegistry, c.DockerSpec.OutputRepository, time.Since(startTimer).Seconds())
}
func performBuild(configuration *RunConfiguration, c *config.Config, r *docker.RegistryCredentials) {
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

	if err := process.Build(r, c, configuration.NexusDownloader, prepper); err != nil {
		var errorMessage string
		if logrus.GetLevel() >= logrus.DebugLevel {
			errorMessage = "Failed to build image: %+v, Terminating"
		} else {
			errorMessage = "Failed to build image: %v, Terminating"
		}
		logrus.Fatalf(errorMessage, err)
	}
}
