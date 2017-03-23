package docker

import (
	"archive/tar"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type DockerClientConfig struct {
	Endpoint string
}

type DockerBuildConfig struct {
	BuildTarget      string
	BuildFolder      string
	ContextTarReader io.Reader
}

type DockerClient struct {
	client client.Client
}

func (d *DockerClient) BuildImage(dockerBuild DockerBuildConfig) error {
	dockerOpt := types.ImageBuildOptions{
		Tags:           []string{dockerBuild.BuildTarget},
		SuppressOutput: false,
	}

	build, err := d.client.ImageBuild(context.Background(), dockerBuild.ContextTarReader, dockerOpt)
	if err != nil {
		return err
	}

	b, _ := ioutil.ReadAll(build.Body)
	fmt.Println(string(b))

	return nil
}

func NewDockerClient(config *DockerClientConfig) (*DockerClient, error) {
	// foreloepig bypasser config biten.
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	return &DockerClient{client: *cli}, nil
}

func CreatContextTarStreamToTarWriter(dockerBase string, writer io.Writer) error {
	baseDir := "./"

	dockerTarWriter := tar.NewWriter(writer)
	defer dockerTarWriter.Close()

	err := filepath.Walk(dockerBase,
		func(path string, info os.FileInfo, errfunc error) error {
			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}

			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, dockerBase))

			if err := dockerTarWriter.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(dockerTarWriter, file)
			return err
		})
	if err != nil {
		return err
	}

	return nil
}

func CreatContextTarStreamReader(dockerBase string) io.ReadCloser {
	r, w := io.Pipe()
	go func() {
		w.CloseWithError(CreatContextTarStreamToTarWriter(dockerBase, w))
	}()
	return r
}
