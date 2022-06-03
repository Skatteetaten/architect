package process

import (
	"context"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"github.com/skatteetaten/architect/v2/pkg/docker"
	"github.com/skatteetaten/architect/v2/pkg/nexus"
	"github.com/skatteetaten/architect/v2/pkg/process/tagger"
	"github.com/skatteetaten/architect/v2/pkg/trace"
	"strings"
)

type Builder interface {
	Build(buildConfig docker.BuildConfig, baseimageLayers *LayerProvider) (*LayerProvider, error)
	Pull(ctx context.Context, buildConfig docker.BuildConfig) (*LayerProvider, error)
	Push(ctx context.Context, buildResult *LayerProvider, tag []string) error
}

func Build(ctx context.Context, pullRegistry docker.Registry, pushRegistry docker.Registry, cfg *config.Config,
	downloader nexus.Downloader, prepper Prepper, builder Builder) error {
	sporingscontext := cfg.SporingsContext
	if sporingscontext == "" {
		logrus.Infof("Use Context %s from the build definition", cfg.OwnerReferenceUid)
		sporingscontext = cfg.OwnerReferenceUid
	}
	tracer := trace.NewTracer(cfg.Sporingstjeneste, sporingscontext)

	logrus.Debugf("Download deliverable for GAV %-v", cfg.ApplicationSpec)
	deliverable, err := downloader.DownloadArtifact(&cfg.ApplicationSpec.MavenGav)
	if err != nil {
		return errors.Wrapf(err, "Could not download deliverable %-v", cfg.ApplicationSpec)
	}
	application := cfg.ApplicationSpec
	logrus.Debug("Extract build info")

	logrus.Infof("Fetching image info %s:%s", application.BaseImageSpec.BaseImage, application.BaseImageSpec.BaseVersion)

	//TODO: Refactor out baseimage-stuff
	imageInfo, err := pullRegistry.GetImageInfo(ctx, application.BaseImageSpec.BaseImage,
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
		application.MavenGav.Version, buildImage, baseImage.DockerImage, deliverable.SHA1)

	dockerBuildConfigArray, err := prepper(cfg, auroraVersion, deliverable, baseImage)
	if err != nil {
		return errors.Wrap(err, "Error preparing image")
	}

	err = checkAllTagsForOverwrite(ctx, dockerBuildConfigArray, pushRegistry, cfg)
	if err != nil {
		return err
	}

	for _, buildConfig := range dockerBuildConfigArray {

		if cfg.NexusIQReportUrl != "" {
			buildConfig.Labels["no.skatteetaten.aurora.nexus-iq-report-url"] = cfg.NexusIQReportUrl
		}

		baseimageLayers, err := builder.Pull(ctx, buildConfig)
		if err != nil {
			return errors.Wrap(err, "There was an error with the pull operation.")
		}

		tracer.AddBaseImageMetadata(application, imageInfo, baseimageLayers.ContainerConfig)

		logrus.Info("Docker context ", buildConfig.BuildFolder)

		dependencyMetadata, _ := nexus.ExtractDependecyMetadata(buildConfig.BuildFolder)

		buildResult, err := builder.Build(buildConfig, baseimageLayers)
		if err != nil {
			return errors.Wrap(err, "There was an error with the build operation.")
		}

		var tagResolver tagger.TagResolver
		if cfg.DockerSpec.TagWith == "" {
			tagResolver = &tagger.NormalTagResolver{
				RegistryClient: pushRegistry,
				Registry:       cfg.DockerSpec.OutputRegistry,
				Repository:     buildConfig.DockerRepository,
			}
		} else {
			tagResolver = &tagger.SingleTagTagResolver{
				Tag:        cfg.DockerSpec.TagWith,
				Registry:   cfg.DockerSpec.OutputRegistry,
				Repository: buildConfig.DockerRepository,
			}
		}

		tags, err := tagResolver.ResolveTags(buildConfig.AuroraVersion, cfg.DockerSpec.PushExtraTags)
		t, err := tagResolver.ResolveShortTag(buildConfig.AuroraVersion, cfg.DockerSpec.PushExtraTags)
		if err != nil {
			return errors.Wrapf(err, "Image tag failed")
		}
		metatags := make(map[string]string)
		for i, tag := range tags {
			metatags[t[i]] = tag
		}
		if !cfg.NoPush {
			err = builder.Push(ctx, buildResult, tags)
			manifest, err := pushRegistry.GetImageInfo(ctx, buildConfig.DockerRepository, t[0])
			if err == nil {
				imageConfig, err := pushRegistry.GetImageConfig(ctx, buildConfig.DockerRepository, manifest.Digest)
				if err == nil {
					tracer.AddImageMetadata(trace.DeployableImage{
						Type:         "deployableImage",
						Digest:       manifest.Digest,
						Name:         buildConfig.DockerRepository,
						Tags:         metatags,
						ImageConfig:  imageConfig,
						NexusSHA1:    deliverable.SHA1,
						Dependencies: dependencyMetadata,
					})
				} else {
					logrus.Warnf("Unable to find information about %s:%s. Got error: %s",
						buildConfig.DockerRepository, imageInfo.Digest, err)
				}
			} else {
				logrus.Warnf("Unable to find information about %s:%s. Got error: %s",
					buildConfig.DockerRepository, t[0], err)
			}
		}
		return err
	}
	return nil
}

func checkAllTagsForOverwrite(ctx context.Context, dockerBuildConfigArray []docker.BuildConfig, pushRegistry docker.Registry, cfg *config.Config) error {
	tagsAPIResponse, err := pushRegistry.GetTags(ctx, cfg.DockerSpec.OutputRepository)
	if err != nil {
		return err
	}
	for _, buildConfig := range dockerBuildConfigArray {
		isSnapshot := buildConfig.AuroraVersion.Snapshot
		tagWith := cfg.DockerSpec.TagWith
		semanticVersion := buildConfig.AuroraVersion.GetGivenVersion()
		completeVersion := buildConfig.AuroraVersion.GetCompleteVersion()
		err = checkTagsForOverwrite(isSnapshot, tagsAPIResponse.Tags, tagWith, semanticVersion, completeVersion)
		if err != nil {
			return err
		}
	}
	return nil
}

/**
Check that we do not overwrite existing TAGS
 	SNAPSHOT tags can be overwritten
*/
func checkTagsForOverwrite(isSnapshot bool, tags []string, tagWith string, semanticVersion string, completeVersion string) error {
	if isSnapshot {
		return nil
	}
	if tagWith != "" && strings.Contains(tagWith, "-SNAPSHOT") {
		return nil
	}
	logrus.Debugf("GivenVersion=%s, CompleteVersion=%s", semanticVersion, completeVersion)
	for _, tag := range tags {
		repoTag := docker.ConvertTagToRepositoryTag(tag)
		logrus.Debug(repoTag)
		if strings.EqualFold(repoTag, completeVersion) {
			return errors.Errorf("There is already a build with tag %s, overwrite not allowed", completeVersion)
		}
		if strings.EqualFold(repoTag, semanticVersion) {
			return errors.Errorf("There is already a build with tag %s, overwrite not allowed", semanticVersion)
		}
		if strings.EqualFold(repoTag, tagWith) {
			return errors.Errorf("Given value for TagWith=%s have already been build, overwrite not allowed", tagWith)
		}
	}
	return nil
}
