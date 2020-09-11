package docker

import (
	"archive/tar"
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

//TODO: Fix context!!!
type RegistryCredentials struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Serveraddress string `json:"serveraddress,omitempty"`
}

type DockerBuildConfig struct {
	AuroraVersion    *runtime.AuroraVersion
	DockerRepository string ///TODO: Refactor? We need to have to different for nodejs
	BuildFolder      string
	Baseimage        runtime.DockerImage //We need to pull the newest image...
	OutputRegistry   string
	Env              map[string]string
	Labels           map[string]string
	Cmd              []string
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

	return filepath.Join(usr.HomeDir, ".docker/config.json"), nil
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
		logrus.Infof("Will not load registry credentials. No entry for %s in %s.. Trying without credentials.",
			outputRegistry, dockerConfigPath)
		return nil, nil
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
