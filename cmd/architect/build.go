package architect

import (
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/docker"
	"github.com/skatteetaten/architect/v2/pkg/nexus"
	"github.com/skatteetaten/architect/v2/pkg/util"
	"github.com/spf13/cobra"
)

var noPush bool

func init() {
	Build.Flags().StringP("file", "f", "", "Path to the compressed leveransepakke")
	Build.Flags().StringP("type", "t", "java", "Application type [java, doozer, nodejs]")
	Build.Flags().StringP("output", "o", "", "Output repository with tag e.g aurora/architect:latest")
	Build.Flags().StringP("from", "", "", "Base image e.g aurora/wingnut11:latest")
	Build.Flags().StringP("push-registry", "", "container-registry-internal.aurora.skead.no", "Push registry")
	Build.Flags().StringP("pull-registry", "", "container-registry-internal-private-pull.aurora.skead.no", "Pull registry")
	Build.Flags().BoolVarP(&noPush, "no-push", "", false, "If true the image is not pushed")
	Build.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	Bc.Flags().StringP("file", "f", "", "Path to a build configuration file")
	Bc.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")

}

// Build command
var Build = &cobra.Command{
	Use:   "build",
	Short: "build file --file <file> --from <baseimage:version> --output <repository:tag> --type [java | nodejs | doozer]",
	Long:  "build images from source",
	Run: func(cmd *cobra.Command, args []string) {

		var nexusDownloader nexus.Downloader
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		} else {
			logrus.SetLevel(logrus.InfoLevel)
		}

		notValid := len(cmd.Flag("file").Value.String()) == 0 ||
			len(cmd.Flag("output").Value.String()) == 0 ||
			len(cmd.Flag("from").Value.String()) == 0 ||
			len(cmd.Flag("type").Value.String()) == 0

		if notValid {
			err := cmd.Help()
			if err != nil {
				panic(err)
			}
			return
		}

		leveransepakke := cmd.Flag("file").Value.String()
		logrus.Debugf("Building %s", leveransepakke)

		// Read build config
		var configReader = config.NewCmdConfigReader(cmd, args, noPush)
		c, err := configReader.ReadConfig()

		if err != nil {
			logrus.Fatalf("Could not read configuration: %s", err)
		}

		var binaryInput string

		binaryInput, err = util.ExtractBinaryFromFile(leveransepakke)
		if err != nil {
			logrus.Fatalf("Could not read binary input: %s", err)
		}

		nexusDownloader = nexus.NewBinaryDownloader(binaryInput)

		RunArchitect(RunConfiguration{
			NexusDownloader:         nexusDownloader,
			Config:                  c,
			RegistryCredentialsFunc: docker.LocalRegistryCredentials(),
		})
	},
}

// Bc build command using buildConfig as input
var Bc = &cobra.Command{
	Use:   "bc",
	Short: "build bc --file <bc>.json",
	Long:  "Build images from openshift build configurations",
	Run: func(cmd *cobra.Command, args []string) {

		var nexusDownloader nexus.Downloader
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		} else {
			logrus.SetLevel(logrus.InfoLevel)
		}

		configPath := cmd.Flag("file").Value.String()
		logrus.Debugf("Building from %s", configPath)

		// Read build config
		var configReader = config.NewFileConfigReader(configPath)

		c, err := configReader.ReadConfig()
		if err != nil {
			logrus.Fatalf("Could not read configuration: %s", err)
		}
		nexusAccess, err := config.ReadNexusAccessFromEnvVars()
		if err != nil {
			logrus.Fatalf("Unable to get Nexus credentials: %s", err)
		}

		nexusDownloader = nexus.NewMavenDownloader(nexusAccess.NexusURL, nexusAccess.Username, nexusAccess.Password)

		RunArchitect(RunConfiguration{
			NexusDownloader:         nexusDownloader,
			Config:                  c,
			RegistryCredentialsFunc: docker.LocalRegistryCredentials(),
		})
	},
}
