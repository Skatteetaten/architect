package docker_test

import (
	"bytes"
	"context"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/docker"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
)

type DockerClientMock struct {
	ImageBuildFunc func(ctx context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error)
	ImagePushFunc  func(ctx context.Context, ref string, options types.ImagePushOptions) (io.ReadCloser, error)
	ImageTagFunc   func(ctx context.Context, image, ref string) error
}

func TestBuildImageSuccess(t *testing.T) {
	target := getBuildTargetFromFile(t, "testdata/rsp_build_success.txt")

	// Uncomment to invoke Docker engine
	//target, _ := docker.NewDockerClient(&docker.DockerClientConfig{Endpoint: ""})

	dir, err := createDockerBase()
	if err != nil {
		t.Error(err)
	}

	if imageid, err := target.BuildImage(dir); err != nil {
		t.Error(err)
	} else if imageid != "6757955c1ca1" {
		t.Errorf("Build returned unexpected image id %s", imageid)
	}
}

func TestBuildImageIllegalDockerfile(t *testing.T) {
	target := getBuildTargetFromFile(t, "testdata/rsp_build_illegal_dockerfile.txt")

	dir, err := createDockerBase()
	if err != nil {
		t.Error(err)
	}

	if _, err = target.BuildImage(dir); err == nil {
		t.Errorf("Expected error")
	} else if !strings.Contains(err.Error(), "Unknown instruction: FOO") {
		t.Errorf("Expected error to contain cause of error")
	}
}

func TestBuildImageError(t *testing.T) {
	target := getBuildTargetError(t)

	dir, err := createDockerBase()

	if err != nil {
		t.Error(err)
	}

	if _, err = target.BuildImage(dir); err == nil {
		t.Errorf("Expected error")
	}
}

func TestPushImageSuccess(t *testing.T) {
	target := getPushTargetFromFile(t, "testdata/rsp_push_success.txt")

	// Uncomment to invoke Docker engine
	//target, _ := docker.NewDockerClient(&docker.DockerClientConfig{Endpoint: ""})

	credentials := docker.RegistryCredentials{}
	err := target.PushImage("foo/bar", &credentials)
	//err := target.PushImage("docker-registry-default.qa.paas.skead.no/aurora/architecttest:1.0.2")

	if err != nil {
		t.Error("Returned unexpected error")
	}
}

func TestPushImageUnauthorized(t *testing.T) {
	target := getPushTargetFromFile(t, "testdata/rsp_push_unauthorized.txt")

	credentials := docker.RegistryCredentials{}
	err := target.PushImage("foo/baz", &credentials)

	if err == nil {
		t.Errorf("Expected error")
	} else if !strings.Contains(err.Error(), "unauthorized: authentication required") {
		t.Errorf("Expected error to contain cause of error")
	}
}

func TestPushImageError(t *testing.T) {
	target := getPushTargetError(t)

	credentials := docker.RegistryCredentials{}
	err := target.PushImage("foo/qux", &credentials)

	if err == nil {
		t.Errorf("Expected error")
	} else if !strings.Contains(err.Error(), "Nasty errror occurred") {
		t.Errorf("Expected error to contain cause of error")
	}
}

func getPushTargetFromFile(t *testing.T, file string) docker.DockerClient {
	body, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err, "Failed to read testdata file")
	}

	return docker.DockerClient{Client: DockerClientMock{ImagePushFunc: func(ctx context.Context, ref string, options types.ImagePushOptions) (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewReader([]byte(body))), nil
	}},
	}
}

func getPushTargetError(t *testing.T) docker.DockerClient {
	return docker.DockerClient{Client: DockerClientMock{ImagePushFunc: func(ctx context.Context, ref string, options types.ImagePushOptions) (io.ReadCloser, error) {
		return nil, errors.New("Nasty errror occurred")
	}},
	}
}

func getBuildTargetFromFile(t *testing.T, file string) docker.DockerClient {
	body, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err, "Failed to read testdata file")
	}

	return docker.DockerClient{Client: DockerClientMock{ImageBuildFunc: func(ctx context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
		return types.ImageBuildResponse{
			Body:   ioutil.NopCloser(bytes.NewReader([]byte(body))),
			OSType: "waldo",
		}, nil
	},
	},
	}
}

func getBuildTargetError(t *testing.T) docker.DockerClient {
	return docker.DockerClient{Client: DockerClientMock{ImageBuildFunc: func(ctx context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
		return types.ImageBuildResponse{}, errors.New("Nasty errror occurred")
	}},
	}
}

func createDockerBase() (string, error) {
	dir, err := ioutil.TempDir("", "dockerbase")
	if err != nil {
		return "", err
	}

	dockerFile, err := os.Create(dir + "/Dockerfile")
	defer dockerFile.Close()
	if err != nil {
		return "", err
	}

	newDir := dir + "/dir1/dir2"
	os.MkdirAll(newDir, 0777)
	echoFileName := "echome.txt"

	t := time.Now()

	echoFile, err := os.Create(newDir + "/" + echoFileName)
	echoFile.WriteString("Date created " + t.String())
	echoFile.Close()

	var dockerFileContent string = `FROM alpine:3.3
		ADD ` + strings.TrimPrefix(newDir, dir) + "/" + echoFileName + ` ./
		RUN cat ` + echoFileName
	dockerFile.WriteString(dockerFileContent)

	return dir, nil
}

func (client DockerClientMock) ImageBuild(ctx context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	return client.ImageBuildFunc(ctx, context, options)
}

func (client DockerClientMock) ImagePush(ctx context.Context, ref string, options types.ImagePushOptions) (io.ReadCloser, error) {
	return client.ImagePushFunc(ctx, ref, options)
}

func (client DockerClientMock) ImageTag(ctx context.Context, image, ref string) error {
	return client.ImageTagFunc(ctx, image, ref)
}
