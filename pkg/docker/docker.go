package docker

import (
	"archive/tar"
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

type RegistryCredentials struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Serveraddress string `json:"serveraddress,omitempty"`
}

type DockerBuildConfig struct {
	Tags        []string
	BuildFolder string
}

type DockerClient struct {
	Client DockerClientAPI
}

func NewDockerClient() (*DockerClient, error) {
	cli, err := client.NewClient(client.DefaultDockerHost, "1.23", nil, nil)

	if err != nil {
		return nil, err
	}

	return &DockerClient{Client: DockerClientProxy{*cli}}, nil
}

func (d *DockerClient) BuildImage(buildConfig DockerBuildConfig) (string, error) {
	dockerOpt := types.ImageBuildOptions{
		Tags:           buildConfig.Tags,
		SuppressOutput: false,
	}
	tarReader := createContextTarStreamReader(buildConfig.BuildFolder)
	build, err := d.Client.ImageBuild(context.Background(), tarReader, dockerOpt)
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

func (d *DockerClient) TagImage(imageId string, tag string) error {
	if err := d.Client.ImageTag(context.Background(), imageId, tag); err != nil {
		return err
	}
	return nil
}

func (d *DockerClient) TagImages(imageId string, tags []string) error {
	for _, tag := range tags {
		err := d.TagImage(imageId, tag)
		if err != nil {
			return errors.Wrap(err, "Error Tagging image")
		}
	}
	return nil
}

func (d *DockerClient) PushImage(tag, credentials string) error {
	logrus.Infof("Pushing image %s", tag)

	pushOptions := createImagePushOptions(credentials)

	push, err := d.Client.ImagePush(context.Background(), tag, pushOptions)

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

func (d *DockerClient) PushImages(tags []string, credentials string) error {
	for _, tag := range tags {
		err := d.PushImage(tag, credentials)
		if err != nil {
			return errors.Wrapf(err, "Failed to push %s", tag)
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

func (n ImageName) String() string {
	if n.Registry == "" {
		return fmt.Sprintf("%s:%s", n.Repository, n.Tag)
	}

	return fmt.Sprintf("%s/%s:%s", n.Registry, n.Repository, n.Tag)
}

func (rc RegistryCredentials) Encode() (string, error) {
	ser, err := json.Marshal(rc)

	if err != nil {
		return "", errors.Wrap(err, "Failed to serialize credentials to json")
	}

	return base64.StdEncoding.EncodeToString(ser), nil
}

func GetDockerConfigPath() (string, error) {
	usr, err := user.Current()

	if err != nil {
		return "", err
	}

	return filepath.Join(usr.HomeDir, ".dockercfg"), nil
}

func createImagePushOptions(credentials string) types.ImagePushOptions {

	if credentials == "" {
		return types.ImagePushOptions{RegistryAuth: "aurora"}
	}

	return types.ImagePushOptions{RegistryAuth: credentials}
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
