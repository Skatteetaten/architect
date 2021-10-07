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
	"net/url"
)

type retagger struct {
	Config       *config.Config
	Credentials  *docker.RegistryCredentials
	PullRegistry docker.Registry
	Builder      process.Builder
}

func newRetagger(cfg *config.Config, credentials *docker.RegistryCredentials, pullRegistry docker.Registry, builder process.Builder) *retagger {
	return &retagger{
		Config:       cfg,
		Credentials:  credentials,
		PullRegistry: pullRegistry,
		Builder:      builder,
	}
}

func Retag(ctx context.Context, cfg *config.Config, credentials *docker.RegistryCredentials, pullRegistry docker.Registry, builder process.Builder) error {
	r := newRetagger(cfg, credentials, pullRegistry, builder)
	return r.Retag(ctx)
}

//TODO: Rewrite this.. This is moving between registries. Check if we need to use the pull registry
func (m *retagger) Retag(ctx context.Context) error {
	tag := m.Config.DockerSpec.RetagWith
	repository := m.Config.DockerSpec.OutputRepository

	logrus.Debug("Get ENV from image manifest")

	imageInfo, err := m.PullRegistry.GetImageInfo(ctx, repository, tag)

	if err != nil {
		return errors.Wrap(err, "Failed to retag image")
	}

	envMap := imageInfo.Enviroment

	// Get AURORA_VERSION
	auroraVersion, ok := envMap[docker.EnvAuroraVersion]

	if !ok {
		return errors.Errorf("Failed to extract ENV variable %s from temporary image manifest", docker.EnvAuroraVersion)
	}

	appVersionString, ok := envMap[docker.EnvAppVersion]

	if !ok {
		return errors.Errorf("Failed to extract ENV variable %s from temporary image manifest", docker.EnvAppVersion)
	}

	givenVersionString, snapshot := envMap[docker.EnvSnapshotVersion]

	if !snapshot {
		givenVersionString = appVersionString
	}

	appVersion := runtime.NewAuroraVersion(appVersionString, snapshot, givenVersionString, runtime.CompleteVersion(auroraVersion))

	extratags, ok := envMap[docker.EnvPushExtraTags]
	if !ok {
		return errors.Errorf("Failed to extract ENV variable %s from temporary image manifest", docker.EnvPushExtraTags)
	}

	pushExtraTags := config.ParseExtraTags(extratags)

	retagRegistryUrl := url.URL{
		Host:   m.Config.DockerSpec.OutputRegistry,
		Scheme: "https",
	}

	//This in only for the push-registry
	retagRegistry := docker.RegistryConnectionInfo{
		Port:        docker.GetPortOrDefault(retagRegistryUrl.Port()),
		Insecure:    docker.InsecureOrDefault(m.Config),
		Host:        retagRegistryUrl.Hostname(),
		Credentials: m.Credentials,
	}
	t := tagger.NormalTagResolver{
		Repository: m.Config.DockerSpec.OutputRepository,
		Registry:   m.Config.DockerSpec.OutputRegistry,
		//TODO: Fix signature.. We don't want to have to registries on retag...
		RegistryClient: docker.NewRegistryClient(retagRegistry),
	}
	logrus.Debugf("Extract tag info, auroraVersion=%v, appVersion=%v, extraTags=%s", auroraVersion, appVersion, extratags)

	tagsToPush, err := t.ResolveTags(appVersion, pushExtraTags)

	if err != nil {
		return err
	}

	//We need to pull to make sure we push the newest image.. We should probably do this directly
	//on the registry when we get v2 registry!:)
	pull := runtime.DockerImage{
		Registry:   m.Config.DockerSpec.GetInternalPullRegistryWithoutProtocol(),
		Repository: m.Config.DockerSpec.OutputRepository,
		Tag:        m.Config.DockerSpec.RetagWith,
	}

	buildConfig := docker.BuildConfig{
		Image: pull,
	}

	imageLayers, err := m.Builder.Pull(ctx, buildConfig)
	if err != nil {
		return errors.Wrapf(err, "Failed to pull image: %v", pull)
	}
	sourceTag := pull.GetCompleteDockerTagName()
	logrus.Debugf("Retagging temporary image, versionTags=%-v", tagsToPush)
	for _, tag := range tagsToPush {
		logrus.Infof("Tag image %s with alias %s", sourceTag, tag)
	}

	err = m.Builder.Push(ctx, imageLayers, tagsToPush)
	if err != nil {
		return errors.Wrapf(err, "Failed to push tag %s", tag)
	}

	return nil
}
