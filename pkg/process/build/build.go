package process

import (
	"context"
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/nexus"
	"github.com/skatteetaten/architect/pkg/process/tagger"
	"github.com/skatteetaten/architect/pkg/trace"
)

type Builder interface {
	Build(ctx context.Context, buildFolder string) (string, error)
	Push(ctx context.Context, imageid string, tag []string, credentials *docker.RegistryCredentials) error
	Tag(ctx context.Context, imageid string, tag string) error
	Pull(ctx context.Context, image runtime.DockerImage) error
}

type metadata struct {
	ImageName    string
	ImageInfo    map[string]interface{}
	Tags         map[string]string
	NexusSHA1    string
	Dependencies []nexus.Dependency
}

func Build(ctx context.Context, credentials *docker.RegistryCredentials, provider docker.ImageInfoProvider, cfg *config.Config, downloader nexus.Downloader, prepper Prepper, builder Builder) error {
	tracer := trace.NewTracer(cfg.Sporingstjeneste, cfg.SporingsContext)
	logrus.Debugf("Download deliverable for GAV %-v", cfg.ApplicationSpec)
	deliverable, err := downloader.DownloadArtifact(&cfg.ApplicationSpec.MavenGav, &cfg.NexusAccess)
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

	baseImageConfig, err := provider.GetImageConfig(application.BaseImageSpec.BaseImage, imageInfo.Digest)
	if err == nil {
		d, err := json.Marshal(baseImageConfig)
		if err == nil {
			tracer.AddImageMetadata("baseImage", "image", string(d))
		}
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

		builder.Pull(ctx, buildConfig.Baseimage)

		logrus.Info("Docker context ", buildConfig.BuildFolder)

		dependencyMetadata, _ := nexus.ExtractDependecyMetadata(buildConfig.BuildFolder)

		imageid, err := builder.Build(ctx, buildConfig.BuildFolder)

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
		t, _ := tagResolver.GetTags(buildConfig.AuroraVersion, cfg.DockerSpec.PushExtraTags)
		metaTags := make(map[string]string)
		for i, tag := range tags {
			err = builder.Tag(ctx, imageid, tag)
			if err != nil {
				return errors.Wrapf(err, "Image tag failed")
			}
			metaTags[t[i]] = tag
		}

		err = builder.Push(ctx, imageid, tags, credentials)

		imageInfo, err := provider.GetImageInfo(buildConfig.DockerRepository, t[0])
		if err == nil {

			imageConfig, err := provider.GetImageConfig(buildConfig.DockerRepository, imageInfo.Digest)
			if err == nil {
				meta := metadata{
					ImageName:    buildConfig.DockerRepository,
					Tags:         metaTags,
					ImageInfo:    imageConfig,
					NexusSHA1:    deliverable.SHA1,
					Dependencies: dependencyMetadata,
				}

				metameta, err := json.Marshal(meta)
				if err == nil {
					tracer.AddImageMetadata("releasedImage", "releasedImage", string(metameta))
				}

			}
		}

		return err
	}
	return nil
}
