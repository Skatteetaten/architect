package docker

import (
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"os"
	"os/user"
	"path/filepath"
)

// RegistryCredentials registry access url and credentials
type RegistryCredentials struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Serveraddress string `json:"serveraddress,omitempty"`
}

// BuildConfig image build configuration
type BuildConfig struct {
	AuroraVersion    *runtime.AuroraVersion
	DockerRepository string
	BuildFolder      string
	Image            runtime.DockerImage //We need to pull the newest image...
	OutputRegistry   string
	Env              map[string]string
	Labels           map[string]string
	Cmd              []string
	Entrypoint       []string
}

// GetDockerConfigPath path to the docker configuration file
func GetDockerConfigPath() (string, error) {
	usr, err := user.Current()

	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, ".docker/config.json"), nil
}

// LocalRegistryCredentials load docker credentials from localhost
func LocalRegistryCredentials() func(string) (*RegistryCredentials, error) {
	return func(outputRegistry string) (*RegistryCredentials, error) {
		dockerConfigPath, err := GetDockerConfigPath()

		if err != nil {
			return nil, err
		}

		return readRegistryCredentials(outputRegistry, dockerConfigPath)
	}
}

// ClusterRegistryCredentials load docker credentials from build pod
func ClusterRegistryCredentials() func(string) (*RegistryCredentials, error) {
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

	dockerConfig, err := readConfig(dockerConfigReader)

	if err != nil {
		return nil, err
	}

	basicCredentials, err := dockerConfig.getCredentials(outputRegistry)

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

	return &registryCredentials, nil
}
