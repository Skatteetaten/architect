package process

import (
	"context"
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

func (d *DockerCmd) Build(ctx context.Context, buildfolder string) (string, error) {
	return d.client.BuildImage(ctx, buildfolder)
}

func (d *DockerCmd) Pull(ctx context.Context, image runtime.DockerImage, credentials *docker.RegistryCredentials) error {
	return d.client.PullImage(ctx, image)
}

func (d *DockerCmd) Tag(ctx context.Context, imageid string, tag string) error {
	return d.client.TagImage(ctx, imageid, tag)
}

func (d *DockerCmd) Push(ctx context.Context, imageid string, tags []string, credentials *docker.RegistryCredentials) error {
	return d.client.PushImages(ctx, tags, credentials)
}
