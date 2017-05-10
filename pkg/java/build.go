package java

import (
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/opencontainers/runc/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/java/nexus"
	"github.com/skatteetaten/architect/pkg/java/prepare"
	"github.com/pkg/errors"
	"strings"
)

func Build(cfg config.Config, downloader nexus.Downloader) error {

	// Download artifact
	deliverable, err := downloader.DownloadArtifact(&cfg.MavenGav)
	if err != nil {
		return errors.Wrap(err,"Could not download artifact")
	}

	// Derive tags
	buildInfo, err := config.NewBuildInfo(docker.NewRegistryClient(cfg.DockerSpec.ExternalDockerRegistry), cfg, *deliverable)
	if err != nil {
		return errors.Wrap(err,"Error in creating buildinfo")
	}

	// Prepare output image
	path, err := prepare.Prepare(*buildInfo, *deliverable)
	if err != nil {
		return errors.Wrap(err,"Error prepare artifact")
	}

	logrus.Infof("Prepare successful. Trigger docker build in %s", path)

	// Build docker image and create tags
	tags := config.GetVersionTags(*buildInfo)
	tagsToPush := createTags(tags, cfg.DockerSpec)
	buildConf := docker.DockerBuildConfig{
		Tags:         tagsToPush,
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

	// Push to registry
	err = client.PushImages(tagsToPush)
	if err != nil {
		return errors.Wrap(err,"Error pushing image")
	}

	return nil
}

func Retag(cfg config.Config) error {

	tag := cfg.DockerSpec.RetagWith
	repository := cfg.DockerSpec.OutputRepository

	manifestProvider := docker.NewRegistryClient(cfg.DockerSpec.ExternalDockerRegistry)

	envMap, err := manifestProvider.GetManifestEnvMap(repository, tag)

	if err != nil {
		return errors.Wrap(err,"Failed to retag image")
	}

	imageId := &docker.ImageName{cfg.DockerSpec.OutputRegistry, cfg.DockerSpec.OutputRepository,
				     cfg.DockerSpec.RetagWith}

	client, err := docker.NewDockerClient(&docker.DockerClientConfig{})
	if err != nil {
		return errors.Wrap(err,"Error initializing Docker")
	}

	// Push tag AURORA_VERSION
	auroraVersion, ok := envMap["AURORA_VERSION"]

	if ! ok {
		return errors.Errorf("Failed to extract ENV variable AURORA_VERSION from temporary image manifest")
	}

	tagAndPushImage(*client, cfg.DockerSpec, imageId.String(), auroraVersion)

	// Push tag APP_VERSION
	appVersion, ok := envMap["APP_VERSION"]

	if ! ok {
		return errors.Errorf("Failed to extract ENV variable APP_VERSION from temporary image manifest")
	}

	tagAndPushImage(*client, cfg.DockerSpec, imageId.String(), appVersion)

	// Push extra tags
	extraTags, ok := envMap["PUSH_EXTRA_TAGS"]

	if ok {
		if strings.Contains(extraTags, "latest") {
			tagAndPushImage(*client, cfg.DockerSpec, imageId.String(), "latest")
		}

		if strings.Contains(extraTags, "major") {
			majorVersion, err := config.GetMajor(appVersion)

			if err != nil {
				return err
			}
			tagAndPushImage(*client, cfg.DockerSpec, imageId.String(), majorVersion)
		}

		if strings.Contains(extraTags, "minor") {
			minorVersion, err := config.GetMinor(appVersion)

			if err != nil {
				return err
			}
			tagAndPushImage(*client, cfg.DockerSpec, imageId.String(), minorVersion)
		}

		if strings.Contains(extraTags, "patch") {
			patchVersion := appVersion

			tagAndPushImage(*client, cfg.DockerSpec, imageId.String(), patchVersion)
		}
	}

	// Create tag SNAPSHOT_TAG
	snapshotTag, ok := envMap["SNAPSHOT_TAG"]

	if ok {
		tagAndPushImage(*client, cfg.DockerSpec, imageId.String(), snapshotTag)
	}

	return nil
}

func tagAndPushImage(client docker.DockerClient, dockerSpec config.DockerSpec, imageId string, tag string) error {
	alias := &docker.ImageName{dockerSpec.OutputRegistry, dockerSpec.OutputRepository,
				       tag}
	err := client.TagImage(imageId, alias.String())

	if err != nil {
		return errors.Wrapf(err, "Failed to tag image %s", imageId)
	}

	err = client.PushImage(alias.String())

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


