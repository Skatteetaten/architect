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

type Builder interface {
	Build(buildFolder string) (string, error)
	Push(imageid string, tag []string, credentials *docker.RegistryCredentials) error
	Tag(imageid string, tag string) error
	Pull(image runtime.DockerImage) error
}

func Build(credentials *docker.RegistryCredentials, provider docker.ImageInfoProvider, cfg *config.Config, downloader nexus.Downloader, prepper Prepper, builder Builder) error {

	logrus.Debugf("Download deliverable for GAV %-v", cfg.ApplicationSpec)
	deliverable, err := downloader.DownloadArtifact(&cfg.ApplicationSpec.MavenGav)
	if err != nil {
		return errors.Wrapf(err, "Could not download deliverable %-v", cfg.ApplicationSpec)
	}
	application := cfg.ApplicationSpec
	logrus.Debug("Extract build info")

	imageInfo, err := provider.GetImageInfo(application.BaseImageSpec.BaseImage,
		application.BaseImageSpec.BaseVersion)
	if err != nil {
		return errors.Wrap(err, "Unable to get the complete build version")
	}

	completeBaseImageVersion := imageInfo.CompleteBaseImageVersion

	baseImage := runtime.BaseImage{
		DockerImage: runtime.DockerImage{
			Tag:        completeBaseImageVersion,
			Repository: application.BaseImageSpec.BaseImage,
			Registry:   cfg.DockerSpec.GetInternalPullRegistryWithoutProtocol(),
		},
		ImageInfo: imageInfo,
	}

	buildImage := &runtime.ArchitectImage{
		Tag: cfg.BuilderSpec.Version,
	}
	snapshot := application.MavenGav.IsSnapshot()
	appVersion := nexus.GetSnapshotTimestampVersion(application.MavenGav, deliverable)
	auroraVersion := runtime.NewAuroraVersionFromBuilderAndBase(appVersion, snapshot,
		application.MavenGav.Version, buildImage, baseImage.DockerImage)

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

	for _, buildConfig := range dockerBuildConfig {

		builder.Pull(buildConfig.Baseimage)

		logrus.Info("Docker context ", buildConfig.BuildFolder)

		imageid, err := builder.Build(buildConfig.BuildFolder)

		if err != nil {
			return errors.Wrap(err, "There was an error with the build operation.")
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
			tagResolver = &tagger.SingleTagTagResolver{
				Tag:        cfg.DockerSpec.TagWith,
				Registry:   cfg.DockerSpec.OutputRegistry,
				Repository: buildConfig.DockerRepository,
			}
		}

		tags, err := tagResolver.ResolveTags(buildConfig.AuroraVersion, cfg.DockerSpec.PushExtraTags)
		logrus.Debugf("Tag image %s with %s", imageid, tags)

		for _, tag := range tags {
			err = builder.Tag(imageid, tag)
			if err != nil {
				return errors.Wrapf(err, "Image tag failed")
			}
		}

		return builder.Push(imageid, tags, credentials)
	}
	return nil
}
