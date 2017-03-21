package docker

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestBuildImage(t *testing.T) {
	cli, err := NewDockerClient(&DockerClientConfig{Endpoint: ""})
	if err != nil {
		t.Error(err)
	}

	dir, err := createDockerBase()
	if err != nil {
		t.Error(err)
	}

	if err := cli.BuildImage(DockerBuildConfig{BuildTarget: "test_image", BuildFolder: dir}); err != nil {
		t.Error(err)
	}
}

func createDockerBase() (string, error) {
	dir, err := ioutil.TempDir("", "dockerbase")
	if err != nil {
		return "", err
	}

	dockerFile, err := os.Create(dir + "/Dockerfile")
	if err != nil {
		return "", err
	}
	var dockerFileContent string = `FROM alpine:3.3
		RUN echo "hello world!"`
	dockerFile.WriteString(dockerFileContent)
	defer dockerFile.Close()

	return dir, nil
}
