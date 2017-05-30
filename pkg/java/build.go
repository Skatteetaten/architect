package java

import (
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/Sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/java/nexus"
	"github.com/skatteetaten/architect/pkg/java/prepare"
	"github.com/pkg/errors"
)

func Build(cfg config.Config, downloader nexus.Downloader) error {

	logrus.Debugf("Download deliverable for GAV %-v", cfg.MavenGav)
	deliverable, err := downloader.DownloadArtifact(&cfg.MavenGav)
	if err != nil {
		return errors.Wrapf(err,"Could not download deliverable %-v", cfg.MavenGav)
	}

	logrus.Debug("Extract build info")
	provider := docker.NewRegistryClient(cfg.DockerSpec.ExternalDockerRegistry)
	buildInfo, err := config.NewBuildInfo(cfg, *deliverable,  provider)
	if err != nil {
		return errors.Wrap(err,"Failed to create buildinfo")
	}

	logrus.Debug("Prepare output image")
	path, err := prepare.Prepare(*buildInfo, *deliverable)

	if err != nil {
		return errors.Wrap(err,"Error prepare artifact")
	}

	versionTags := buildInfo.OutputImage.VersionTags

	if !cfg.DockerSpec.TagOverwrite {
		logrus.Debug("Tags Overwrite diabled, filtering tags")

		repositoryTags, err := provider.GetTags(cfg.DockerSpec.OutputRepository)

		if err != nil {
			return errors.Wrap(err,"Error in GetTags")
		}

		versionTags, err = config.FilterVersionTags(buildInfo.Env[docker.ENV_APP_VERSION], versionTags, repositoryTags.Tags)

		if err != nil {
			return errors.Wrap(err,"Error in FilterTags")
		}
	}

	logrus.Debugf("Build docker image and create tags, path=%s, buildInfo=%-v", path, *buildInfo)
	tagsToPush := createTags(versionTags, cfg.DockerSpec)

	buildConf := docker.DockerBuildConfig{
		Tags:        tagsToPush,
		BuildFolder: path,
	}

	client, err := docker.NewDockerClient(&docker.DockerClientConfig{})

	if err != nil {
		return errors.Wrap(err,"Error initializing Docker")
	}

	imageid, err := client.BuildImage(buildConf)

	if err != nil {
		return errors.Wrap(err,"Fuckup!")
	} else {
		logrus.Infof("Done building. Imageid: %s", imageid)
	}

	logrus.Debug("Push images and tags")
	err = client.PushImages(tagsToPush)
	if err != nil {
		return errors.Wrap(err,"Error pushing images")
	}

	return nil
}

func Retag(cfg config.Config) error {
	tag := cfg.DockerSpec.RetagWith
	repository := cfg.DockerSpec.OutputRepository

	logrus.Debug("Get ENV from image manifest")
	manifestProvider := docker.NewRegistryClient(cfg.DockerSpec.ExternalDockerRegistry)

	envMap, err := manifestProvider.GetManifestEnvMap(repository, tag)

	if err != nil {
		return errors.Wrap(err,"Failed to retag image")
	}

	client, err := docker.NewDockerClient(&docker.DockerClientConfig{})
	if err != nil {
		return errors.Wrap(err,"Error initializing Docker")
	}

	// Get AURORA_VERSION
	auroraVersion, ok := envMap[docker.ENV_AURORA_VERSION]

	if ! ok {
		return errors.Errorf("Failed to extract ENV variable %s from temporary image manifest", docker.ENV_AURORA_VERSION)
	}

	appVersion, ok := envMap[docker.ENV_APP_VERSION]

	if ! ok {
		return errors.Errorf("Failed to extract ENV variable %s from temporary image manifest", docker.ENV_APP_VERSION)
	}

	extratags, ok := envMap[docker.ENV_PUSH_EXTRA_TAGS]

	if ! ok {
		return errors.Errorf("Failed to extract ENV variable %s from temporary image manifest", docker.ENV_PUSH_EXTRA_TAGS)
	}

	logrus.Debugf("Extract tag info, auroraVersion=%s, appVersion=%s, extraTags=%s", auroraVersion, appVersion, extratags)
	var tagInfo *config.TagInfo

	provider := docker.NewRegistryClient(cfg.DockerSpec.ExternalDockerRegistry)

	tagInfo, err = config.NewTagInfo(appVersion, auroraVersion, extratags)

	imageId := &docker.ImageName{cfg.DockerSpec.OutputRegistry, cfg.DockerSpec.OutputRepository,
				     cfg.DockerSpec.RetagWith}

	versionTags := tagInfo.VersionTags

	if !cfg.DockerSpec.TagOverwrite {
		logrus.Debug("Tags Overwrite diabled, filtering tags")

		repositoryTags, err := provider.GetTags(cfg.DockerSpec.OutputRepository)

		if err != nil {
			return errors.Wrap(err,"Error in GetTags")
		}

		versionTags, err = config.FilterVersionTags(appVersion, versionTags, repositoryTags.Tags)

		if err != nil {
			return errors.Wrap(err,"Error in FilterTags")
		}
	}


	tagsToPush := createTags(versionTags, cfg.DockerSpec)

	logrus.Debugf("Retag temporary image, versionTags=%-v", *tagInfo)
	for _, alias := range tagsToPush {
		err := tagAndPushImage(*client, cfg.DockerSpec, imageId.String(), alias)

		if err != nil {
			return errors.Wrap(err, "Failed to tag image")
		}
	}

	return nil
}

func tagAndPushImage(client docker.DockerClient, dockerSpec config.DockerSpec, imageId string, alias string) error {

	logrus.Infof("Tag image %s with alias %s", imageId, alias)

	err := client.TagImage(imageId, alias)

	if err != nil {
		return errors.Wrapf(err, "Failed to tag image %s with alias %s", imageId, alias)
	}

	logrus.Infof("Push tag %s to registry", alias)

	err = client.PushImage(alias)

	if err != nil {
		return errors.Wrapf(err, "Failed to push tag %s", alias)
	}

	return nil
}

func createTags(tags []string, dockerSpec config.DockerSpec) []string {
	output := make([]string, len(tags))
	for i, t := range tags {
		name := &docker.ImageName{dockerSpec.OutputRegistry,
					  dockerSpec.OutputRepository, t}
		output[i] = name.String()
	}
	return output
}


