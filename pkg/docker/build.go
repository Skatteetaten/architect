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

type DockerBuild struct {
	BuildFolder string
	BuildTarget string
}

func (dockerBuild *DockerBuild) BuildImage() (string, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	dockerFilePath, err := createDockerTar(dockerBuild.BuildFolder)
	if err != nil {
		return "", nil
	}

	dockerFile, err := os.Open(dockerFilePath)
	if err != nil {
		return "", nil
	}
	defer dockerFile.Close()

	dockerOpt := types.ImageBuildOptions{
		Tags:           []string{"somepathandname"},
		Context:        dockerFile,
		SuppressOutput: false,
	}

	build, err := cli.ImageBuild(context.Background(), dockerFile, dockerOpt)
	if err != nil {
		return "", nil
	}

	b, _ := ioutil.ReadAll(build.Body)
	fmt.Println(string(b))

	return "shouldreturnimageid", nil
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
