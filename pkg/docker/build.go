package docker

import (
	"archive/tar"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io/ioutil"
	"os"
)

type DockerClientConfig struct {
	Endpoint string
}

type DockerBuildConfig struct {
	BuildTarget string
	BuildFolder string
}

type DockerClient struct {
	client client.Client
}

func (d *DockerClient) BuildImage(dockerBuild DockerBuildConfig) error {
	dockerFilePath, err := createDockerTar(dockerBuild.BuildFolder)
	if err != nil {
		return nil
	}

	dockerFile, err := os.Open(dockerFilePath)
	if err != nil {
		return nil
	}
	defer dockerFile.Close()

	dockerOpt := types.ImageBuildOptions{
		Tags:           []string{dockerBuild.BuildTarget},
		Context:        dockerFile,
		SuppressOutput: false,
	}

	build, err := d.client.ImageBuild(context.Background(), dockerFile, dockerOpt)
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

func createDockerTar(dockerBase string) (string, error) {
	dockerTarFile, err := os.Create(os.TempDir() + "/DockerPackage.tar")
	defer dockerTarFile.Close()

	if err != nil {
		return "", err
	}

	dockerTarWriter := tar.NewWriter(dockerTarFile)
	defer dockerTarWriter.Close()

	readDockerBasePath, err := ioutil.ReadDir(dockerBase)

	if err != nil {
		return "", err
	}

	for _, dockerFile := range readDockerBasePath {
		dockerFileContent, err := ioutil.ReadFile(dockerBase + "/" + dockerFile.Name())
		dockerFileHeader, err := tar.FileInfoHeader(dockerFile, "")
		if err != nil {
			return "", err
		}
		dockerTarWriter.WriteHeader(dockerFileHeader)
		dockerTarWriter.Write(dockerFileContent)
	}
	dockerTarWriter.Flush()

	return dockerTarFile.Name(), nil
}
