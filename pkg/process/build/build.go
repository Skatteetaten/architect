package process

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/process/tagger"
)

func Build(credentials *docker.RegistryCredentials, cfg *config.Config, prepper Prepper) error {
	provider := docker.NewRegistryClient(cfg.DockerSpec.ExternalDockerRegistry)
	dockerBuildConfig, err := prepper(cfg, provider)
	if err != nil {
		return errors.Wrap(err, "Error preparing image")
	}

	client, err := docker.NewDockerClient()
	if err != nil {
		return errors.Wrap(err, "Error initializing Docker")
	}

	for _, buildConfig := range dockerBuildConfig {
		imageid, err := client.BuildImage(buildConfig.BuildFolder)

		if err != nil {
			return errors.Wrap(err, "Fuckup!")
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
			tagResolver = &tagger.TagForRetagTagResolver{
				Tag:        cfg.DockerSpec.TagWith,
				Registry:   cfg.DockerSpec.OutputRegistry,
				Repository: buildConfig.DockerRepository,
			}
		}

		tags, err := tagResolver.ResolveTags(buildConfig.AuroraVersion, cfg.DockerSpec.PushExtraTags)
		logrus.Debug("Push images and tags")
		for _, tag := range tags {
			err = client.TagImage(imageid, tag)
			if err != nil {
				return err
			}
		}
		err = client.PushImages(tags, credentials)
		if err != nil {
			return errors.Wrap(err, "Error pushing images")
		}
	}
	return nil
}
