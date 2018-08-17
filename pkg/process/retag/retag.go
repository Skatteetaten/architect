package retag

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
)

type retagger struct {
	Config      *config.Config
	Credentials *docker.RegistryCredentials
}

func newRetagger(cfg *config.Config, credentials *docker.RegistryCredentials) *retagger {
	return &retagger{
		Config:      cfg,
		Credentials: credentials,
	}
}

func Retag(cfg *config.Config, credentials *docker.RegistryCredentials) error {
	r := newRetagger(cfg, credentials)
	return r.Retag()
}

func (m *retagger) Retag() error {
	tag := m.Config.DockerSpec.RetagWith
	repository := m.Config.DockerSpec.OutputRepository

	logrus.Debug("Get ENV from image manifest")
	manifestProvider := docker.NewRegistryClient(m.Config.DockerSpec.ExternalDockerRegistry)

	imageInfo, err := manifestProvider.GetImageInfo(repository, tag)

	if err != nil {
		return errors.Wrap(err, "Failed to retag image")
	}

	envMap := imageInfo.Enviroment

	// Get AURORA_VERSION
	auroraVersion, ok := envMap[docker.ENV_AURORA_VERSION]

	if !ok {
		return errors.Errorf("Failed to extract ENV variable %s from temporary image manifest", docker.ENV_AURORA_VERSION)
	}

	appVersionString, ok := envMap[docker.ENV_APP_VERSION]

	if !ok {
		return errors.Errorf("Failed to extract ENV variable %s from temporary image manifest", docker.ENV_APP_VERSION)
	}

	givenVersionString, snapshot := envMap[docker.ENV_SNAPSHOT_TAG]

	if !snapshot {
		givenVersionString = appVersionString
	}

	appVersion := runtime.NewAuroraVersion(appVersionString, snapshot, givenVersionString, runtime.CompleteVersion(auroraVersion))

	extratags, ok := envMap[docker.ENV_PUSH_EXTRA_TAGS]

	if !ok {
		return errors.Errorf("Failed to extract ENV variable %s from temporary image manifest", docker.ENV_PUSH_EXTRA_TAGS)
	}

	logrus.Debugf("Extract tag info, auroraVersion=%s, appVersion=%s, extraTags=%s", auroraVersion, appVersion, extratags)

	provider := docker.NewRegistryClient(m.Config.DockerSpec.ExternalDockerRegistry)

	if err != nil {
		return errors.Wrap(err, "Unable to get version tags")
	}

	imageId := runtime.DockerImage{
		Registry:   m.Config.DockerSpec.OutputRegistry,
		Repository: m.Config.DockerSpec.OutputRepository,
		Tag:        m.Config.DockerSpec.RetagWith,
	}

	var repositoryTags []string

	if !m.Config.DockerSpec.TagOverwrite {
		logrus.Debug("Tags Overwrite diabled, filtering tags")

		rt, err := provider.GetTags(m.Config.DockerSpec.OutputRepository)

		if err != nil {
			return errors.Wrapf(err, "Error in GetTags, repository=%s", m.Config.DockerSpec.OutputRepository)

		}
		repositoryTags = rt.Tags

	}

	versionTags, err := appVersion.GetApplicationVersionTagsToPush(repositoryTags, config.ParseExtraTags(extratags))

	if err != nil {
		return err
	}

	tagsToPush := docker.CreateImageNameFromSpecAndTags(versionTags,
		m.Config.DockerSpec.OutputRegistry,
		m.Config.DockerSpec.OutputRepository)

	client, err := docker.NewDockerClient()
	if err != nil {
		return errors.Wrap(err, "Error initializing Docker")
	}

	//We need to pull to make sure we push the newest image.. We should probably do this directly
	//on the registry when we get v2 registry!:)
	client.PullImage(imageId)

	logrus.Debugf("Retagging temporary image, versionTags=%-v", versionTags)
	for _, tag := range tagsToPush {
		sourceTag := imageId.GetCompleteDockerTagName()
		logrus.Infof("Tag image %s with alias %s", sourceTag, tag)
		err := client.TagImage(sourceTag, tag)
		if err != nil {
			return errors.Wrapf(err, "Failed to tag image %s with tag %s", imageId, tag)
		}
	}
	for _, tag := range tagsToPush {
		err = client.PushImage(tag, m.Credentials)
		if err != nil {
			return errors.Wrapf(err, "Failed to push tag %s", tag)
		}
	}
	return nil
}
