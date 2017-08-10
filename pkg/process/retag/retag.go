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

	envMap, err := manifestProvider.GetManifestEnvMap(repository, tag)

	if err != nil {
		return errors.Wrap(err, "Failed to retag image")
	}

	client, err := docker.NewDockerClient()
	if err != nil {
		return errors.Wrap(err, "Error initializing Docker")
	}

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

	appVersion := runtime.NewApplicationVersion(appVersionString, snapshot, givenVersionString, runtime.CompleteVersion(auroraVersion))

	extratags, ok := envMap[docker.ENV_PUSH_EXTRA_TAGS]

	if !ok {
		return errors.Errorf("Failed to extract ENV variable %s from temporary image manifest", docker.ENV_PUSH_EXTRA_TAGS)
	}

	logrus.Debugf("Extract tag info, auroraVersion=%s, appVersion=%s, extraTags=%s", auroraVersion, appVersion, extratags)

	provider := docker.NewRegistryClient(m.Config.DockerSpec.ExternalDockerRegistry)

	if err != nil {
		return errors.Wrap(err, "Unable to get version tags")
	}

	imageId := &docker.ImageName{
		Registry:   m.Config.DockerSpec.OutputRegistry,
		Repository: m.Config.DockerSpec.OutputRepository,
		Tag:        m.Config.DockerSpec.RetagWith,
	}

	var repositoryTags *docker.TagsAPIResponse

	if !m.Config.DockerSpec.TagOverwrite {
		logrus.Debug("Tags Overwrite diabled, filtering tags")

		repositoryTags, err = provider.GetTags(m.Config.DockerSpec.OutputRepository)

		if err != nil {
			return errors.Wrapf(err, "Error in GetTags, repository=%s", m.Config.DockerSpec.OutputRepository)

		}

	}

	versionTags, err := appVersion.GetApplicationVersionTagsToPush(repositoryTags.Tags, config.ParseExtraTags(extratags))

	if err != nil {
		return err
	}

	tagsToPush := docker.CreateImageNameFromSpecAndTags(versionTags,
		m.Config.DockerSpec.OutputRegistry,
		m.Config.DockerSpec.OutputRepository)

	logrus.Debugf("Retag temporary image, versionTags=%-v", versionTags)
	for _, alias := range tagsToPush {
		err := m.tagAndPushImage(*client, imageId.String(), alias)

		if err != nil {
			return errors.Wrap(err, "Failed to tag image")
		}
	}

	return nil
}

func (m *retagger) tagAndPushImage(client docker.DockerClient, imageId string, alias string) error {

	logrus.Infof("Tag image %s with alias %s", imageId, alias)

	err := client.TagImage(imageId, alias)

	if err != nil {
		return errors.Wrapf(err, "Failed to tag image %s with alias %s", imageId, alias)
	}

	logrus.Infof("Push tag %s to registry", alias)

	err = client.PushImage(alias, m.Credentials)

	if err != nil {
		return errors.Wrapf(err, "Failed to push tag %s", alias)
	}

	return nil
}
