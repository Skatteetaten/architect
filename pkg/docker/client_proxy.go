package docker

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
)

type DockerClientAPI interface {
	ImageBuild(ctx context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error)
	ImagePush(ctx context.Context, ref string, options types.ImagePushOptions) (io.ReadCloser, error)
	ImageTag(ctx context.Context, image string, ref string) error
	ImagePull(ctx context.Context, image string, options types.ImagePullOptions) (io.ReadCloser, error)
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

func (proxy DockerClientProxy) ImageTag(ctx context.Context, image string, ref string) error {
	return proxy.client.ImageTag(ctx, image, ref)
}

func (proxy DockerClientProxy) ImagePull(ctx context.Context, image string, options types.ImagePullOptions) (io.ReadCloser, error) {
	return proxy.client.ImagePull(ctx, image, options)
}
