package docker

import (
	"io"
	"github.com/docker/docker/client"
	"context"
	"github.com/docker/docker/api/types"
)

type DockerClientAPI interface {
	ImageBuild(ctx context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error)
	ImagePush(ctx context.Context, ref string, options types.ImagePushOptions) (io.ReadCloser, error)
	ImageTag(ctx context.Context, image, ref string) error
}

type DockerClientProxy struct {
	client client.Client
}

func (proxy DockerClientProxy) ImageBuild(ctx context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	return proxy.client.ImageBuild(ctx, context, options)
}

func (proxy DockerClientProxy) ImagePush(ctx context.Context, ref string, options types.ImagePushOptions) (io.ReadCloser, error) {
	return proxy.client.ImagePush(ctx, ref, options)
}

func (proxy DockerClientProxy) ImageTag(ctx context.Context, image, ref string) error {
	return proxy.client.ImageTag(ctx, image, ref)
}
