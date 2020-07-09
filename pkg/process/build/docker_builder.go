package process

import (
	"context"
	"github.com/pkg/errors"
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

func (d *DockerCmd) Build(ctx context.Context, buildConfig docker.DockerBuildConfig) (*BuildOutput, error) {
	imageid, err := d.client.BuildImage(ctx, buildConfig.BuildFolder)
	return &BuildOutput{ImageId: imageid}, err
}

func (d *DockerCmd) Pull(ctx context.Context, buildConfig docker.DockerBuildConfig) error {
	//Buildah dont require this method. better way ?
	return d.client.PullImage(ctx, buildConfig.Baseimage)
}

func (d *DockerCmd) Tag(ctx context.Context, buildResult *BuildOutput, tag string) error {
	imageid := buildResult.ImageId
	return d.client.TagImage(ctx, imageid, tag)
}

func (d *DockerCmd) Push(ctx context.Context, buildResult *BuildOutput, tags []string, credentials *docker.RegistryCredentials) error {
	return d.client.PushImages(ctx, tags, credentials)
}
