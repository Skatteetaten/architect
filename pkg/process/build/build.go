package process

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/nexus"
	"github.com/skatteetaten/architect/pkg/process/tagger"
)

//TODO: Write some test for this..
// Need to initialize RegistryClient and DockerClient outside of this function
func Build(credentials *docker.RegistryCredentials, cfg *config.Config, downloader nexus.Downloader, prepper Prepper) error {
	provider := docker.NewRegistryClient(cfg.DockerSpec.ExternalDockerRegistry)

	logrus.Debugf("Download deliverable for GAV %-v", cfg.ApplicationSpec)
	deliverable, err := downloader.DownloadArtifact(&cfg.ApplicationSpec.MavenGav)
	if err != nil {
		return errors.Wrapf(err, "Could not download deliverable %-v", cfg.ApplicationSpec)
	}
	application := cfg.ApplicationSpec
	logrus.Debug("Extract build info")

	completeBaseImageVersion, err := provider.GetCompleteBaseImageVersion(application.BaseImageSpec.BaseImage,
		application.BaseImageSpec.BaseVersion)
	if err != nil {
		return errors.Wrap(err, "Unable to get the complete build version")
	}

	baseImage := runtime.DockerImage{
		Tag:        completeBaseImageVersion,
		Repository: application.BaseImageSpec.BaseImage,
		Registry:   cfg.DockerSpec.GetExternalRegistryWithoutProtocol(),
	}

	buildImage := &runtime.ArchitectImage{
		Tag: cfg.BuilderSpec.Version,
	}
	snapshot := application.MavenGav.IsSnapshot()
	appVersion := nexus.GetSnapshotTimestampVersion(application.MavenGav, deliverable)
	auroraVersion := runtime.NewAuroraVersionFromBuilderAndBase(appVersion, snapshot,
		application.MavenGav.Version, buildImage, baseImage)
	if err != nil {
		return errors.Wrap(err, "Error creating version information")
	}

	dockerBuildConfig, err := prepper(cfg, auroraVersion, deliverable, baseImage)
	if err != nil {
		return errors.Wrap(err, "Error preparing image")
	}

	if !cfg.DockerSpec.TagOverwrite {
		for _, buildConfig := range dockerBuildConfig {
			if !buildConfig.AuroraVersion.Snapshot {
				tags, err := provider.GetTags(cfg.DockerSpec.OutputRepository)
				if err != nil {
					return err
				}
				completeVersion := buildConfig.AuroraVersion.GetCompleteVersion()
				for _, tag := range tags.Tags {
					if tag == completeVersion {
						return errors.Errorf("There are already a build with tag %s, consider TAG_OVERWRITE", completeVersion)
					}
				}
			}
		}
	}

	client, err := docker.NewDockerClient()
	if err != nil {
		return errors.Wrap(err, "Error initializing Docker")
	}

	for _, buildConfig := range dockerBuildConfig {
		client.PullImage(buildConfig.Baseimage)
		imageid, err := client.BuildImage(buildConfig.BuildFolder)

		if err != nil {
			return errors.Wrap(err, "Fuckup!")
		} else {
			logrus.Infof("Done building. Imageid: %s", imageid)
		}

		var tagResolver tagger.TagResolver
		if cfg.DockerSpec.TagWith == "" {
			tagResolver = &tagger.NormalTagResolver{
				Overwrite:  cfg.DockerSpec.TagOverwrite,
				Provider:   provider,
				Registry:   cfg.DockerSpec.OutputRegistry,
				Repository: buildConfig.DockerRepository,
			}
		} else {
			tagResolver = &tagger.TagForRetagTagResolver{
				Tag:        cfg.DockerSpec.TagWith,
				Registry:   cfg.DockerSpec.OutputRegistry,
				Repository: buildConfig.DockerRepository,
			}
		}

		tags, err := tagResolver.ResolveTags(buildConfig.AuroraVersion, cfg.DockerSpec.PushExtraTags)
		logrus.Debugf("Tag image %s with %s", imageid, tags)
		for _, tag := range tags {
			err = client.TagImage(imageid, tag)
			if err != nil {
				return err
			}
		}
		err = client.PushImages(tags, credentials)
		if err != nil {
			return errors.Wrap(err, "Error pushing images")
		}
	}
	return nil
}
