package process

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/docker"
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
		imageid, err := client.BuildImage(buildConfig)

		if err != nil {
			return errors.Wrap(err, "Fuckup!")
		} else {
			logrus.Infof("Done building. Imageid: %s", imageid)
		}

		logrus.Debug("Push images and tags")
		err = client.PushImages(buildConfig.Tags, credentials)
		if err != nil {
			return errors.Wrap(err, "Error pushing images")
		}
	}
	return nil
}
