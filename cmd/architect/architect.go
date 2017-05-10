package architect

import (
	"github.com/Sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/java/nexus"
	"github.com/skatteetaten/architect/pkg/java/prepare"
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
	c, err := configReader.ReadConfig()
	if err != nil {
		logrus.Fatalf("Could not read configuration: %s", err)
	}
	logrus.Debugf("Config %+v", c)

	if c.DockerSpec.RetagWith != "" {
		retag(*c)
	} else {
		build(*c, downloader)
	}

}

func build(cfg config.Config, downloader nexus.Downloader) {
	deliverable, err := downloader.DownloadArtifact(&cfg.MavenGav)
	if err != nil {
		logrus.Fatalf("Could not download artifact: %s", err)
	}

	buildInfo, err := config.NewBuildInfo(docker.NewRegistryClient(cfg.DockerSpec.ExternalDockerRegistry), cfg, *deliverable)
	if err != nil {
		logrus.Fatalf("Error in creating buildinfo: %s", err)
	}

	path, err := prepare.Prepare(*buildInfo, *deliverable)
	if err != nil {
		logrus.Fatalf("Error prepare artifact: %s", err)
	}

	logrus.Infof("Prepare successful. Trigger docker build in %s", path)

	tags := config.GetVersionTags(*buildInfo)
	tagsToPush := createTags(tags, cfg.DockerSpec)
	buildConf := docker.DockerBuildConfig{
		Tags:         tagsToPush,
		BuildFolder: path,
	}
	client, err := docker.NewDockerClient(&docker.DockerClientConfig{})
	if err != nil {
		logrus.Fatalf("Error initializing Docker", err)
	}
	imageid, err := client.BuildImage(buildConf)

	if err != nil {
		logrus.Fatalf("Fuckup! %+v", err)
	} else {
		logrus.Infof("Done building. Imageid: %s", imageid)
	}
	err = client.PushImages(tagsToPush)
	if err != nil {
		logrus.Fatalf("Error pushing image %+v", err)
	}
}

func retag(config config.Config) {
	
}


func createTags(tags []string, dockerSpec config.DockerSpec) []string {
	output := make([]string, len(tags))
	for i, t := range tags {
		output[i] = dockerSpec.OutputRegistry + "/" + dockerSpec.OutputRepository + ":" + t
	}
	return output
}
