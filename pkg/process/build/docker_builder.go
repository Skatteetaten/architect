package process

import (
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
)

func NewDockerBuilder() (*DockerCmd, error) {

	client, err := docker.NewDockerClient()
	if err != nil {
		return nil, errors.Wrap(err, "Error initializing Docker Client")
	}
	return &DockerCmd{
		client: client,
	}, nil
}

type DockerCmd struct {
	client *docker.DockerClient
}

func (d *DockerCmd) Build(buildfolder string) (string, error) {
	return d.client.BuildImage(buildfolder)
}

func (d *DockerCmd) Pull(image runtime.DockerImage) error {
	//Buildah dont require this method. better way ?
	return d.client.PullImage(image)
}

func (d *DockerCmd) Tag(imageid string, tag string) error {
	return d.client.TagImage(imageid, tag)
}

func (d *DockerCmd) Push(imageid string, tags []string, credentials *docker.RegistryCredentials) error {
	return d.client.PushImages(tags, credentials)
}
