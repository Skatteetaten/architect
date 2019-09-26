package architect

import (
	"github.com/Sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/nexus"
	"github.com/skatteetaten/architect/pkg/util"
	"github.com/spf13/cobra"
)

func init() {
	Build.Flags().StringP("fileconfig", "f", "", "Path to file config. If not set, the environment variable BUILD is read")
	Build.Flags().StringP("skippush", "s", "", "If set, Docker push will not be performed")
	Build.Flags().BoolVarP(&localRepo, "binary", "b", false, "If set, the Leveransepakke will be fetched from stdin")
	Build.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
}

var Build = &cobra.Command{

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
			mavenRepo := "https://aurora/nexus/service/local/artifact/maven/content"
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
