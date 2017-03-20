package config

import (
	"os"
	"errors"
	"encoding/json"
	"github.com/skatteetaten/architect/pkg/config/api"
	"io/ioutil"
)

type ConfigReader interface {
	ReadConfig() (*Config, error)
}

type InClusterConfigReader struct {
}

type FileConfigReader struct {
	pathToConfigFile string
}

func NewInClusterConfigReader() ConfigReader {
	return &InClusterConfigReader{}
}

func NewFileConfigReader(filepath string) ConfigReader {
	return &FileConfigReader{pathToConfigFile: filepath}
}

func (m *FileConfigReader) ReadConfig() (*Config, error) {
	dat, err := ioutil.ReadFile(m.pathToConfigFile)
	if err != nil {
		return nil, err;
	}
	return newConfig(dat)

}

func (m *InClusterConfigReader) ReadConfig() (*Config, error) {
	buildConfig := os.Getenv("BUILD")
	if len(buildConfig) == 0 {
		return nil, errors.New("Expected a build config environment variable to be present.")
	}
	return newConfig([]byte(buildConfig))

}

func newConfig(buildConfig []byte) (*Config, error) {
	build := api.Build{}
	err := json.Unmarshal(buildConfig, &build)
	if err != nil {
		return nil, err
	}
	customStrategy := build.Spec.Strategy.CustomStrategy
	if customStrategy == nil {
		return nil, errors.New("Expected strategy to be custom strategy. Thats the only one supported.")
	}
	gav := NexusGav{}
	if artifactId, err := findEnv(customStrategy.Env, "ARTIFACT_ID"); err == nil {
		gav.ArtifactId = artifactId
	} else {
		return nil, err
	}
	if groupId, err := findEnv(customStrategy.Env, "GROUP_ID"); err == nil {
		gav.GroupId = groupId
	} else {
		return nil, err
	}
	if version, err := findEnv(customStrategy.Env, "VERSION"); err == nil {
		gav.Version = version
	} else {
		return nil, err
	}

	dockerSpec := DockerSpec{}
	outputKind := build.Spec.Output.To.Kind
	if outputKind != "DockerImage" {
		return nil, errors.New("This image only supports output of kind DockerImage")
	}
	output := build.Spec.Output.To.Name;
	dockerSpec.Registry = output
	c := &Config{
		ApplicationType: JavaLeveransepakke,
		NexusGav: gav,
		DockerSpec: dockerSpec,
	}
	return c, nil;
}

func findEnv(vars []api.EnvVar, name string) (string, error) {
	for _, e := range vars {
		if e.Name == name {
			return e.Value, nil
		}
	}
	return "", errors.New("No env variable with name " + name)
}

