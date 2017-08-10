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
	"github.com/skatteetaten/architect/pkg/config/runtime"
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
	AuroraVersion    *runtime.AuroraVersion
	DockerRepository string ///TODO: Refactor? We need to have to different for nodejs
	BuildFolder      string
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

func (d *DockerClient) BuildImage(buildFolder string) (string, error) {
	dockerOpt := types.ImageBuildOptions{
		SuppressOutput: false,
	}
	tarReader := createContextTarStreamReader(buildFolder)
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

func (d *DockerClient) PushImage(tag string, credentials *RegistryCredentials) error {
	logrus.Infof("Pushing image %s", tag)

	var encodedCredentials string
	if credentials == nil {
		encodedCredentials = ""
	} else {
		c, err := credentials.Encode()
		if err != nil {
			return errors.Wrap(err, "Unable to create credentials")
		}
		encodedCredentials = c
	}
	pushOptions := createImagePushOptions(encodedCredentials)

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

func (d *DockerClient) PushImages(tags []string, credentials *RegistryCredentials) error {
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

func LocalRegistryCredentials() func(string) (*RegistryCredentials, error) {
	return func(outputRegistry string) (*RegistryCredentials, error) {
		dockerConfigPath, err := GetDockerConfigPath()

		if err != nil {
			return nil, err
		}

		return readRegistryCredentials(outputRegistry, dockerConfigPath)
	}
}

func CusterRegistryCredentials() func(string) (*RegistryCredentials, error) {
	return func(outputRegistry string) (*RegistryCredentials, error) {
		return readRegistryCredentials(outputRegistry, "/var/run/secrets/openshift.io/push/.dockercfg")
	}
}

func readRegistryCredentials(outputRegistry string, dockerConfigPath string) (*RegistryCredentials, error) {
	_, err := os.Stat(dockerConfigPath)

	if err != nil {
		if os.IsNotExist(err) {
			logrus.Infof("Will not load registry credentials. %s not found.", dockerConfigPath)
			return nil, nil
		}

		return nil, err
	}

	dockerConfigReader, err := os.Open(dockerConfigPath)

	if err != nil {
		return nil, err
	}

	dockerConfig, err := ReadConfig(dockerConfigReader)

	if err != nil {
		return nil, err
	}

	basicCredentials, err := dockerConfig.GetCredentials(outputRegistry)

	if err != nil {
		return nil, err
	} else if basicCredentials == nil {
		logrus.Infof("Will not load registry credentials. No entry for %s in %s.", outputRegistry, dockerConfigPath)
		return nil, errors.Errorf("No credentials found for registry " + outputRegistry)
	}

	registryCredentials := RegistryCredentials{
		basicCredentials.User,
		basicCredentials.Password,
		outputRegistry,
	}

	if err != nil {
		return nil, err
	}

	return &registryCredentials, nil
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
