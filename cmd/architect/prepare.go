package architect

import (
	"github.com/Sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config"
	java "github.com/skatteetaten/architect/pkg/java/prepare"
	"github.com/skatteetaten/architect/pkg/nexus"
	"github.com/spf13/cobra"
)

func init() {
	Prepare.Flags().StringP("workspace", "w", "", "Path to workspace.")
	Prepare.Flags().StringP("artifact", "a", "", "Artifact")
	Prepare.Flags().StringP("group", "g", "", "Group id")
	Prepare.Flags().StringP("version", "v", "", "Version")
	Prepare.Flags().BoolVarP(&verbose, "verbose", "", false, "Verbose logging")
}

var Prepare = &cobra.Command{

	Use:   "prepare",
	Short: "Prepare Docker context",
	Long:  `TODO`,
	Run: func(cmd *cobra.Command, args []string) {

		configReader := config.NewAuroraBuildConfigReader()
		c, err := configReader.ReadConfig()
		if err != nil {
			logrus.Fatalf("Could not read configuration: %s", err)
		}

		logrus.Debugf("Config: %v", c)

		var nexusDownloader nexus.Downloader
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		} else {
			logrus.SetLevel(logrus.InfoLevel)
		}
		if len(cmd.Flag("workspace").Value.String()) == 0 {
			logrus.Error("Specify workspace")
			return
		}

		workspace := cmd.Flag("workspace").Value.String()

		mavenRepo := "http://aurora/nexus/service/local/artifact/maven/content"
		logrus.Debugf("Using Maven repo on %s", mavenRepo)
		nexusDownloader = nexus.NewPathAwareNexusDownloader(mavenRepo, workspace)

		gav := c.ApplicationSpec.MavenGav

		deliverble, err := nexusDownloader.DownloadArtifact(&gav)


		if err != nil {
			logrus.Error("Artifact downloaded failed ", err)
		}

		//Generate docker files (radish for now)
		_, err = java.PrepareFiles(workspace, deliverble, c)

		if err != nil {
			logrus.Error("Prepare files failed ", err)
		}

	},
}
