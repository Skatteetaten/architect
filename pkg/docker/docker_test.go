package docker_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
	"io"
	"github.com/docker/docker/api/types"
	"golang.org/x/net/context"
	"github.com/skatteetaten/architect/pkg/docker"
	"bytes"
	"github.com/pkg/errors"
)

type DockerClientMock struct {
	ImageBuildFunc func(ctx context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error)
	ImagePushFunc func(ctx context.Context, ref string, options types.ImagePushOptions) (io.ReadCloser, error)
	ImageTagFunc func(ctx context.Context, image, ref string) error
}

var ImagePushSuccess = func(ctx context.Context, ref string, options types.ImagePushOptions) (io.ReadCloser, error) {
	body := `{"status":"The push refers to a repository [registry09:5000/foo/huey]"}
{"status":"Preparing","progressDetail":{},"id":"363011c5287c"}
{"status":"Waiting","progressDetail":{},"id":"dbed221c3f7b"}
{"progressDetail":{},"aux":{"Tag":"tag","Digest":"sha256:0ce54ead","Size":2611}}`

	return ioutil.NopCloser(bytes.NewReader([]byte(body))), nil
}

var ImagePushUnauthorized = func(ctx context.Context, ref string, options types.ImagePushOptions) (io.ReadCloser, error) {
	body := `{"status":"The push refers to a repository [registry09:5000/foo/dewey]"}
{"status":"Preparing","progressDetail":{},"id":"363011c5287c"}
{"status":"Waiting","progressDetail":{},"id":"dbed221c3f7b"}
{"status":"Image push failed","progressDetail":{},"id":"363011c5287c"}
{"errorDetail":{"message":"unauthorized: authentication required"},"error":"unauthorized: authentication required"}`

	return ioutil.NopCloser(bytes.NewReader([]byte(body))), nil
}

var ImagePushError = func(ctx context.Context, ref string, options types.ImagePushOptions) (io.ReadCloser, error) {
	return nil, errors.New("Nasty errror occurred")
}

func TestBuildImage(t *testing.T) {
	cli, err := docker.NewDockerClient(&docker.DockerClientConfig{Endpoint: ""})
	if err != nil {
		t.Error(err)
	}

	dir, err := createDockerBase()
	if err != nil {
		t.Error(err)
	}

	buildConfig := docker.DockerBuildConfig{
		Tags:        []string{"test_image"},
		BuildFolder: dir,
	}

	imageid, err := cli.BuildImage(buildConfig)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println("Image id created: " + imageid)
	}
}

func TestPushImageSuccess(t *testing.T) {
	target := docker.DockerClient{Client: DockerClientMock{ImagePushFunc: ImagePushSuccess}}
	err := target.PushImage("foo/huey")

	if err != nil {
		t.Errorf("Returned unexpected error")
	}
}

func TestPushImageUnauthortized(t *testing.T) {
	target := docker.DockerClient{Client: DockerClientMock{ImagePushFunc: ImagePushUnauthorized}}
	err := target.PushImage("foo/dewey")

	if err == nil {
		t.Errorf("Expected error")
	} else if ! strings.Contains(err.Error(), "unauthorized: authentication required") {
		t.Errorf("Expected error to contain cause of error")
	}
}
func TestPushImageError(t *testing.T) {
	target := docker.DockerClient{Client: DockerClientMock{ImagePushFunc: ImagePushError}}
	err := target.PushImage("foo/louie")

	if err == nil {
		t.Errorf("Expected error")
	} else if ! strings.Contains(err.Error(), "Nasty errror occurred") {
		t.Errorf("Expected error to contain cause of error")
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
