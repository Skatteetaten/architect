package retag

import (
	"context"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	process "github.com/skatteetaten/architect/pkg/process/build"
	"github.com/skatteetaten/architect/pkg/process/tagger"
)

type retagger struct {
	Config      *config.Config
	Credentials *docker.RegistryCredentials
	Provider    docker.ImageInfoProvider
	Builder     process.Builder
}

func newRetagger(cfg *config.Config, credentials *docker.RegistryCredentials, provider docker.ImageInfoProvider, builder process.Builder) *retagger {
	return &retagger{
		Config:      cfg,
		Credentials: credentials,
		Provider:    provider,
		Builder:     builder,
	}
}

func Retag(ctx context.Context, cfg *config.Config, credentials *docker.RegistryCredentials, provider docker.ImageInfoProvider, builder process.Builder) error {
	r := newRetagger(cfg, credentials, provider, builder)
	return r.Retag(ctx)
}

func (m *retagger) Retag(ctx context.Context) error {
	tag := m.Config.DockerSpec.RetagWith
	repository := m.Config.DockerSpec.OutputRepository

	logrus.Debug("Get ENV from image manifest")

	imageInfo, err := m.Provider.GetImageInfo(repository, tag)

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

	pushExtraTags := config.ParseExtraTags(extratags)

	t := tagger.NormalTagResolver{
		Repository: m.Config.DockerSpec.OutputRepository,
		Registry:   m.Config.DockerSpec.OutputRegistry,
		Provider:   docker.NewRegistryClient(m.Config.DockerSpec.ExternalDockerRegistry),
		Overwrite:  m.Config.DockerSpec.TagOverwrite,
	}
	logrus.Debugf("Extract tag info, auroraVersion=%v, appVersion=%v, extraTags=%s", auroraVersion, appVersion, extratags)

	if err != nil {
		return errors.Wrap(err, "Unable to get version tags")
	}

	push := runtime.DockerImage{
		Registry:   m.Config.DockerSpec.OutputRegistry,
		Repository: m.Config.DockerSpec.OutputRepository,
		Tag:        m.Config.DockerSpec.RetagWith,
	}

	pull := runtime.DockerImage{
		Registry:   m.Config.DockerSpec.GetInternalPullRegistryWithoutProtocol(),
		Repository: m.Config.DockerSpec.OutputRepository,
		Tag:        m.Config.DockerSpec.RetagWith,
	}

	tagsToPush, err := t.ResolveTags(appVersion, pushExtraTags)

	if err != nil {
		return err
	}

	if err != nil {
		return errors.Wrap(err, "Error initializing Docker")
	}

	//We need to pull to make sure we push the newest image.. We should probably do this directly
	//on the registry when we get v2 registry!:)
	err = m.Builder.Pull(ctx, pull, m.Credentials)
	if err != nil {
		return errors.Wrapf(err, "Failed to pull image: %v", pull)
	}
	sourceTag := pull.GetCompleteDockerTagName()
	logrus.Debugf("Retagging temporary image, versionTags=%-v", tagsToPush)
	for _, tag := range tagsToPush {
		logrus.Infof("Tag image %s with alias %s", sourceTag, tag)
		err := m.Builder.Tag(ctx, sourceTag, tag)
		if err != nil {
			return errors.Wrapf(err, "Failed to tag image %s with tag %s", push, tag)
		}
	}

	err = m.Builder.Push(ctx, sourceTag, tagsToPush, m.Credentials)
	if err != nil {
		return errors.Wrapf(err, "Failed to push tag %s", tag)
	}

	return nil
}
