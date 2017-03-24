package docker

import (
	"archive/tar"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
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

func (d *DockerClient) BuildImage(buildConfig DockerBuildConfig) (string, error) {
	dockerOpt := types.ImageBuildOptions{
		Tags:           []string{buildConfig.BuildTarget},
		SuppressOutput: false,
	}

	build, err := d.client.ImageBuild(context.Background(), buildConfig.ContextTarReader, dockerOpt)
	if err != nil {
		return "", err
	}

	// ImageBuild will not return error message if build fails.
	var bodyLine string = ""
	scanner := bufio.NewScanner(build.Body)
	for scanner.Scan() {
		bodyLine = scanner.Text()
		if strings.Contains(bodyLine, "errorDetail") {
			msg, err := JsonMapToString(bodyLine, "error")
			if err != nil {
				return "", err
			}
			return "", errors.New(msg)
		}
	}
	// Get image id.
	msg, err := JsonMapToString(bodyLine, "stream")

	return strings.TrimPrefix(msg, "Successfully built "), nil
}

func JsonMapToString(jsonStr string, key string) (string, error) {
	var f interface{}
	if err := json.Unmarshal([]byte(jsonStr), &f); err != nil {
		return "", err
	}
	errorMap := f.(map[string]interface{})
	return errorMap[key].(string), nil
}

func NewDockerClient(config *DockerClientConfig) (*DockerClient, error) {
	// foreloepig bypasser config biten.
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	return &DockerClient{client: *cli}, nil
}

func CreateContextTarStreamToTarWriter(dockerBase string, writer io.Writer) error {
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

func CreateContextTarStreamReader(dockerBase string) io.ReadCloser {
	r, w := io.Pipe()
	go func() {
		w.CloseWithError(CreateContextTarStreamToTarWriter(dockerBase, w))
	}()
	return r
}
