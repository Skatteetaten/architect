package architect

import (
	"github.com/Sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/java"
	"github.com/skatteetaten/architect/pkg/java/nexus"
	"github.com/skatteetaten/architect/pkg/nodejs/npm"
	"github.com/skatteetaten/architect/pkg/nodejs/prepare"
	"github.com/skatteetaten/architect/pkg/process/build"
	"github.com/skatteetaten/architect/pkg/process/retag"
	"github.com/skatteetaten/architect/pkg/util"
	"github.com/spf13/cobra"
)

var localRepo bool
var verbose bool

type RunConfiguration struct {
	NexusDownloader         nexus.Downloader
	NpmDownloader           npm.Downloader
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
		var npmDownloader npm.Downloader
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
				logrus.Fatalf("Could not read binary input", err)
			}
		}

		if localRepo {
			logrus.Debugf("Using local maven repo")
			nexusDownloader = nexus.NewLocalDownloader()
			npmDownloader = npm.NewLocalRegistry(".")
		} else if c.BinaryBuild {
			nexusDownloader = nexus.NewLocalDownloader() //TODO: Wont work with binary input
			npmDownloader = npm.NewBinaryBuildRegistry(binaryInput, c.NodeJsApplication.Version)
		} else {
			mavenRepo := "http://aurora/nexus/service/local/artifact/maven/content"
			logrus.Debugf("Using Maven repo on %s", mavenRepo)
			nexusDownloader = nexus.NewNexusDownloader(mavenRepo)
			npmDownloader = npm.NewRemoteRegistry("http://aurora/npm/repository/npm-internal/")
		}

		RunArchitect(RunConfiguration{
			NexusDownloader:         nexusDownloader,
			Config:                  c,
			NpmDownloader:           npmDownloader,
			RegistryCredentialsFunc: docker.LocalRegistryCredentials(),
		})
	},
}

func init() {
	JavaLeveransepakke.Flags().StringP("fileconfig", "f", "", "Path to file config. If not set, the environment variable BUILD is read")
	JavaLeveransepakke.Flags().StringP("skippush", "s", "", "If set, Docker push will not be performed")
	JavaLeveransepakke.Flags().BoolVarP(&localRepo, "localrepo", "l", false, "If set, the Leveransepakke will be fetched from the local repo")
	JavaLeveransepakke.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
}

func RunArchitect(configuration RunConfiguration) {
	c := configuration.Config
	logrus.Debugf("Config %+v", c)

	registryCredentials, err := configuration.RegistryCredentialsFunc(c.DockerSpec.OutputRegistry)

	if err != nil {
		logrus.Fatal("Cound not parse registry credentials", err)
	}

	if c.DockerSpec.RetagWith != "" {
		logrus.Info("Perform retag")
		if err := retag.Retag(c, registryCredentials); err != nil {
			logrus.Fatal("Failed to retag temporary image", err)
		}
	} else {
		performBuild(&configuration, c, registryCredentials)

	}

}
func performBuild(configuration *RunConfiguration, c *config.Config, r *docker.RegistryCredentials) {
	var prepper process.Prepper
	if c.ApplicationType == config.JavaLeveransepakke {
		logrus.Info("Perform Java build")
		prepper = java.Prepper(configuration.NexusDownloader)

	} else if c.ApplicationType == config.NodeJsLeveransepakke {
		logrus.Info("Perform Webleveranse build")
		prepper = prepare.Prepper(configuration.NpmDownloader)
	}

	if err := process.Build(r, c, prepper); err != nil {
		logrus.Fatalf("Failed to build image: %s", err)
	}
}
