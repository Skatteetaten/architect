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

// Builder interface
type Builder interface {
	Build(buildConfig docker.BuildConfig, baseimageLayers *LayerProvider) (*LayerProvider, error)
	Pull(ctx context.Context, buildConfig docker.BuildConfig) (*LayerProvider, error)
	Push(ctx context.Context, buildResult *LayerProvider, tag []string) error
}

// Build a container image
func Build(ctx context.Context, pullRegistry docker.Registry, pushRegistry docker.Registry, cfg *config.Config,
	downloader nexus.Downloader, prepper Prepper, layerBuilder Builder, sporingsLoggerClient trace.Trace) error {
	application := cfg.ApplicationSpec
	snapshot := application.MavenGav.IsSnapshot()
	buildImage := &runtime.ArchitectImage{
		Tag: cfg.BuilderSpec.Version,
	}
	deliverable, err := downloader.DownloadArtifact(&application.MavenGav)
	if err != nil {
		return errors.Wrapf(err, "Could not download deliverable %-v", cfg.ApplicationSpec)
	}

	baseImage, err := getBaseImage(ctx, pullRegistry, err, cfg)
	if err != nil {
		return errors.Wrap(err, "Error getBaseImage")
	}

	appVersion := nexus.GetSnapshotTimestampVersion(application.MavenGav, deliverable)
	auroraVersion := runtime.NewAuroraVersionFromBuilderAndBase(appVersion, snapshot,
		application.MavenGav.Version, buildImage, baseImage.DockerImage, deliverable.SHA1)

	logrus.Infof("appversion %s  auroraVersion:%s ", appVersion, auroraVersion.GetCompleteVersion())
	logrus.Infof(" MavenGav.Version:%s", application.MavenGav.Version)

	dockerBuildConfig, err := prepper(cfg, auroraVersion, deliverable, baseImage)
	if err != nil {
		return errors.Wrap(err, "Error preparing image")
	}

	err = checkAllTagsForOverwrite(ctx, *dockerBuildConfig, pushRegistry, cfg)
	if err != nil {
		return err
	}

	tags, shortTags, err := extractTags(*dockerBuildConfig, pushRegistry, cfg)
	if err != nil {
		return errors.Wrapf(err, "Unable to extract tags")
	}

	buildResult, err := buildDockerImage(ctx, *dockerBuildConfig, cfg, layerBuilder)
	if err != nil {
		return errors.Wrap(err, "There was an error with the build operation.")
	}

	err = pushImage(ctx, cfg, buildResult, layerBuilder, tags)
	if err != nil {
		return errors.Wrapf(err, "Image push failed")
	}

	err = sendImageInfoToSporingsLogger(ctx, cfg, dockerBuildConfig.DockerRepository, pushRegistry, sporingsLoggerClient, auroraVersion, shortTags)
	if err != nil {
		logrus.Warnf("Unable to send trace to Sporinglogger  %s:%s  error: %v",
			dockerBuildConfig.DockerRepository, shortTags[0], err)
		return nil
	}

	return nil
}

func getBaseImage(ctx context.Context, pullRegistry docker.Registry, err error, cfg *config.Config) (runtime.BaseImage, error) {
	baseImageSpec := cfg.ApplicationSpec.BaseImageSpec
	logrus.Infof("Fetching image info %s:%s", baseImageSpec.BaseImage, baseImageSpec.BaseVersion)

	imageInfo, err := pullRegistry.GetImageInfo(ctx, baseImageSpec.BaseImage,
		baseImageSpec.BaseVersion)
	if err != nil {
		return runtime.BaseImage{}, errors.Wrap(err, "Unable to get the complete build version")
	}

	completeBaseImageVersion := imageInfo.CompleteBaseImageVersion

	baseImage := runtime.BaseImage{
		DockerImage: runtime.DockerImage{
			Tag:        completeBaseImageVersion,
			Repository: baseImageSpec.BaseImage,
			Registry:   cfg.DockerSpec.GetInternalPullRegistryWithoutProtocol(),
		},
		ImageInfo: imageInfo,
	}
	return baseImage, nil
}

func buildDockerImage(ctx context.Context, buildConfig docker.BuildConfig, cfg *config.Config, layerBuilder Builder) (*LayerProvider, error) {

	if cfg.NexusIQReportURL != "" {
		buildConfig.Labels["no.skatteetaten.aurora.nexus-iq-report-url"] = cfg.NexusIQReportURL
	}

	baseImageLayers, err := layerBuilder.Pull(ctx, buildConfig)
	if err != nil {
		return nil, errors.Wrap(err, "There was an error with the pull operation.")
	}

	return layerBuilder.Build(buildConfig, baseImageLayers)
}
func pushImage(ctx context.Context, cfg *config.Config, buildResult *LayerProvider, layerBuilder Builder, tags []string) error {
	if cfg.NoPush {
		logrus.Info("NoPush configured, not pushing image")
		return nil
	}

	err := layerBuilder.Push(ctx, buildResult, tags)
	if err != nil {
		return errors.Wrapf(err, "Image push failed")
	}

	return nil
}

func sendImageInfoToSporingsLogger(ctx context.Context, cfg *config.Config, serviceName string,
	dockerRegistry docker.Registry, sporingsLoggerClient trace.Trace, auroraVersion *runtime.AuroraVersion, shortTags []string) error {
	if cfg.NoPush {
		logrus.Info("NoPush configured, not sending image info to Sporingslogger")
		return nil
	}

	imageInfo, err := dockerRegistry.GetImageInfo(ctx, serviceName, shortTags[0])
	if err != nil {
		return errors.Wrapf(err, "Unable to GetImageInfo ")
	}
	logrus.Infof("Sending image info to sporingslogger %s ", imageInfo.Digest)

	return sporingsLoggerClient.SendImageMetadata(trace.DeployableImage{
		Type:          "deployableImage",
		Name:          serviceName,
		AppVersion:    string(auroraVersion.GetAppVersion()),
		AuroraVersion: auroraVersion.GetCompleteVersion(),
		Digest:        imageInfo.Digest,
		Snapshot:      auroraVersion.Snapshot,
	})
}

func checkAllTagsForOverwrite(ctx context.Context, buildConfig docker.BuildConfig, pushRegistry docker.Registry, cfg *config.Config) error {
	tagsAPIResponse, err := pushRegistry.GetTags(ctx, cfg.DockerSpec.OutputRepository)
	if err != nil {
		return err
	}

	isSnapshot := buildConfig.AuroraVersion.Snapshot
	tagWith := cfg.DockerSpec.TagWith
	semanticVersion := buildConfig.AuroraVersion.GetGivenVersion()
	completeVersion := buildConfig.AuroraVersion.GetCompleteVersion()
	err = CheckTagsForOverwrite(isSnapshot, tagsAPIResponse.Tags, tagWith, semanticVersion, completeVersion)

	return err
}

// CheckTagsForOverwrite /
// Check that we do not overwrite existing TAGS
// SNAPSHOT tags can be overwritten
func CheckTagsForOverwrite(isSnapshot bool, tags []string, tagWith string, semanticVersion string, completeVersion string) error {
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

func extractTags(buildConfig docker.BuildConfig, pushRegistry docker.Registry, cfg *config.Config) ([]string, []string, error) {

	var tagResolver tagger.TagResolver
	if cfg.DockerSpec.TagWith == "" {
		tagResolver = &tagger.NormalTagResolver{
			RegistryClient: pushRegistry,
			Registry:       cfg.DockerSpec.OutputRegistry,
			Repository:     buildConfig.DockerRepository,
		}
	} else {
		tagResolver = &tagger.SingleTagResolver{
			Tag:        cfg.DockerSpec.TagWith,
			Registry:   cfg.DockerSpec.OutputRegistry,
			Repository: buildConfig.DockerRepository,
		}
	}

	tags, err := tagResolver.ResolveTags(buildConfig.AuroraVersion, cfg.DockerSpec.PushExtraTags)
	shortTags, err := tagResolver.ResolveShortTag(buildConfig.AuroraVersion, cfg.DockerSpec.PushExtraTags)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Image tag failed")
	}
	return tags, shortTags, nil
}
