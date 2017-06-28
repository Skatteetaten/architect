package architect

import (
	"github.com/Sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/java"
	"github.com/skatteetaten/architect/pkg/java/nexus"
	"github.com/spf13/cobra"
)

var localRepo bool
var verbose bool

var JavaLeveransepakke = &cobra.Command{

	Use:   "build",
	Short: "Build Docker image from Zip",
	Long:  `TODO`,
	Run: func(cmd *cobra.Command, args []string) {
		var configReader = config.NewInClusterConfigReader()
		var downloader nexus.Downloader
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
		if localRepo {
			logrus.Debugf("Using local maven repo")
			downloader = nexus.NewLocalDownloader()
		} else {
			mavenRepo := "http://aurora/nexus/service/local/artifact/maven/content"
			logrus.Debugf("Using Maven repo on %s", mavenRepo)
			downloader = nexus.NewNexusDownloader(mavenRepo)
		}

		RunArchitect(configReader, downloader)
	},
}

func init() {
	JavaLeveransepakke.Flags().StringP("fileconfig", "f", "", "Path to file config. If not set, the environment variable BUILD is read")
	JavaLeveransepakke.Flags().StringP("skippush", "s", "", "If set, Docker push will not be performed")
	JavaLeveransepakke.Flags().BoolVarP(&localRepo, "localrepo", "l", false, "If set, the Leveransepakke will be fetched from the local repo")
	JavaLeveransepakke.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
}

func RunArchitect(configReader config.ConfigReader, downloader nexus.Downloader) {

	// Read build config
	c, err := configReader.ReadConfig()
	if err != nil {
		logrus.Fatalf("Could not read configuration: %s", err)
	}

	logrus.Debugf("Config %+v", c)

	err = configReader.AddRegistryCredentials(c)
	if err != nil {
		logrus.Fatalf("Could not read configuration: %s", err)
	}

	if c.DockerSpec.RetagWith != "" {
		logrus.Info("Perform retag")
		if err := java.Retag(*c); err != nil {
			logrus.Fatalf("Failed to retag temporary image: %s", err)
		}

	} else {
		logrus.Info("Perform build")
		if err := java.Build(*c, downloader); err != nil {
			logrus.Fatalf("Failed to build image: %s", err)
		}
	}

}
