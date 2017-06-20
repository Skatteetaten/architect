package docker

import (
	"archive/tar"
	"bufio"
	"context"
	"encoding/json"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
	"os"
	"path/filepath"
	"strings"
	"github.com/pkg/errors"
	"github.com/Sirupsen/logrus"
)

type DockerClientConfig struct {
	Endpoint string
}

type DockerBuildConfig struct {
	Tags        []string
	BuildFolder string
}

type DockerClient struct {
	client client.Client
}

func (d *DockerClient) BuildImage(buildConfig DockerBuildConfig) (string, error) {
	dockerOpt := types.ImageBuildOptions{
		Tags:           buildConfig.Tags,
		SuppressOutput: false,
	}
	tarReader := createContextTarStreamReader(buildConfig.BuildFolder)
	build, err := d.client.ImageBuild(context.Background(), tarReader, dockerOpt)
	if err != nil {
		return "", errors.Wrap(err, "Error building image")
	}

	// ImageBuild will not return error message if build fails.
	var bodyLine string = ""
	scanner := bufio.NewScanner(build.Body)
	for scanner.Scan() {
		bodyLine = scanner.Text()
		logrus.Debug(bodyLine)
		if strings.Contains(bodyLine, "errorDetail") {
			msg, err := JsonMapToString(bodyLine, "error")
			if err != nil {
				return "", errors.Wrap(err, "Error mapping JSON error message. Error in build.")
			}
			return "", errors.New(msg)
		}
	}
	// Get image id.
	msg, err := JsonMapToString(bodyLine, "stream")

	return strings.TrimSpace(strings.TrimPrefix(msg, "Successfully built ")), nil
}

func (d *DockerClient) TagImages(imageId string, tags[] string) (error) {
	for _, tag := range tags {
		err := d.TagImage(imageId, tag)
		if err != nil {
			return errors.Wrap(err, "Error Tagging image")
		}
	}
	return nil
}

func (d *DockerClient) PushImages(tags[] string) (error) {
	for _, tag := range tags {
		err := d.PushImage(tag)
		if err != nil {
			return errors.Wrapf(err, "Failed to push %s", tag)
		}
	}
	return nil
}

func (d *DockerClient) TagImage(imageId string, tag string) (error) {
	if err := d.client.ImageTag(context.Background(), imageId, tag); err != nil {
		return err
	}
	return nil
}

func (d *DockerClient) PushImage(tag string) (error) {
	logrus.Infof("Pushing image %s", tag)
	push, err := d.client.ImagePush(context.Background(), tag, types.ImagePushOptions{RegistryAuth: "aurora"})

	if err != nil {
		return err
	}

	defer push.Close()

	// ImageBuild will not return error message if build fails.
	scanner := bufio.NewScanner(push)
	for scanner.Scan() {
		bodyLine := scanner.Text()
		logrus.Debug(bodyLine)
		if strings.Contains(bodyLine, "errorDetail") {
			msg, err := JsonMapToString(bodyLine, "error")
			if err != nil {
				return errors.Wrap(err, "Error mapping JSON error message. Unknown error")
			}
			return errors.New(msg)
		}
	}

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
	cli, err := client.NewClient(client.DefaultDockerHost, "1.23", nil, nil)
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
			if info.Mode() & os.ModeSymlink == os.ModeSymlink {
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

			if !info.Mode().IsRegular() {
				//nothing more to do for non-regular
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
