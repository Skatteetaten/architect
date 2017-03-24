package docker

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
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

	tarReader := CreateContextTarStreamReader(dir)
	buildConfig := DockerBuildConfig{
		BuildTarget:      "test_image",
		BuildFolder:      dir,
		ContextTarReader: tarReader,
	}

	imageid, err := cli.BuildImage(buildConfig)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println("Image id created: " + imageid)
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
