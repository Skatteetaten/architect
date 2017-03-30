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
	Tag         string
	BuildFolder string
}

type DockerClient struct {
	client client.Client
}

func (d *DockerClient) BuildImage(buildConfig DockerBuildConfig) (string, error) {
	dockerOpt := types.ImageBuildOptions{
		Tags:           []string{buildConfig.Tag},
		SuppressOutput: false,
	}
	tarReader := createContextTarStreamReader(buildConfig.BuildFolder)
	build, err := d.client.ImageBuild(context.Background(), tarReader, dockerOpt)
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

func (d *DockerClient) PushImage(ImageID string) (error) {
	/*
	Ble lagt til en test for aa sjekke gyldighet til navnet
	Var usikker rundt digest id-en, om denne kunne/skulle pushes. Saa fant en test paa det ogsaa
	Foreloepig styrer vi dette fra andre steder.
	*/
	/*ref, err := reference.ParseNamed(buildConfig.ImageName)
	if err != nil {
		return err
	}

	switch x := ref.(type) {
	case reference.Canonical:
		return errors.New("cannot push a digest reference: " + x.Digest().String())
	}*/

	//push, err := d.client.ImagePush(context.Background(), ref.Name(), types.ImagePushOptions{})
	push, err := d.client.ImagePush(context.Background(), ImageID, types.ImagePushOptions{})
	if err != nil {
		return err
	}
	defer push.Close()

	return nil
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
	cli, err := client.NewClient(client.DefaultDockerHost, "1.24", nil, nil)
	if err != nil {
		return nil, err
	}
	return &DockerClient{client: *cli}, nil
}

func createContextTarStreamToTarWriter(dockerBase string, writer io.Writer) error {
	baseDir := "./"

	dockerTarWriter := tar.NewWriter(writer)
	defer dockerTarWriter.Close()

	err := filepath.Walk(dockerBase,
		func(path string, info os.FileInfo, errfunc error) error {

			var link string
			if info.Mode()&os.ModeSymlink == os.ModeSymlink {
				var err error
				if link, err = os.Readlink(path); err != nil {
					return err
				}
			}

			header, err := tar.FileInfoHeader(info, link)
			if err != nil {
				return err
			}

			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, dockerBase))

			if err := dockerTarWriter.WriteHeader(header); err != nil {
				return err
			}

			if !info.Mode().IsRegular() { //nothing more to do for non-regular
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

func createContextTarStreamReader(dockerBase string) io.ReadCloser {
	r, w := io.Pipe()
	go func() {
		w.CloseWithError(createContextTarStreamToTarWriter(dockerBase, w))
	}()
	return r
}
