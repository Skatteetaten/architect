package config

import (
	"encoding/json"
	"github.com/docker/docker/reference"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config/api"
	"io/ioutil"
	"os"
	"strings"
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
		return nil, err
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
	gav := MavenGav{}
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

	if version, err := findEnv(customStrategy.Env, "VERSION"); err == nil {
		gav.Version = version
	} else {
		return nil, err
	}

	dockerSpec := DockerSpec{}

	if baseImage, err := findEnv(customStrategy.Env, "DOCKER_BASE_IMAGE"); err == nil {
		dockerSpec.BaseImage = baseImage
	} else if baseImage, err := findEnv(customStrategy.Env, "DOCKER_BASE_NAME"); err == nil {
		dockerSpec.BaseImage = baseImage
	} else {
		return nil, err
	}

	if externalRegistry, err := findEnv(customStrategy.Env, "BASE_IMAGE_REGISTRY"); err == nil {
		if strings.HasPrefix(externalRegistry, "https://") {
			dockerSpec.ExternalDockerRegistry = externalRegistry
		} else {
			dockerSpec.ExternalDockerRegistry = "https://" + externalRegistry
		}
	} else {
		dockerSpec.ExternalDockerRegistry = "https://docker-registry.aurora.sits.no:5000"
	}

	if baseImageVersion, err := findEnv(customStrategy.Env, "DOCKER_BASE_VERSION"); err == nil {
		dockerSpec.BaseVersion = baseImageVersion
	} else {
		return nil, err
	}

	builderSpec := BuilderSpec{}

	if builderVersion, err := findEnv(customStrategy.Env, "BUILDER_VERSION"); err == nil {
		builderSpec.Version = builderVersion
	} else {
		builderSpec.Version = "dsjkfl"
		//return nil, err
	}

	outputKind := build.Spec.Output.To.Kind
	if outputKind != "DockerImage" {
		return nil, errors.New("This image only supports output of kind DockerImage")
	}
	output := build.Spec.Output.To.Name

	dockerSpec.OutputRegistry, err = findOutputRegistry(output)
	if err != nil {
		return nil, err
	}
	dockerSpec.OutputRepository, err = findOutputRepository(output)
	if err != nil {
		return nil, err
	}
	c := &Config{
		ApplicationType: JavaLeveransepakke,
		MavenGav:        gav,
		DockerSpec:      dockerSpec,
		BuilderSpec:     builderSpec,
	}
	return c, nil
}

func findOutputRepository(dockerName string) (string, error) {
	name, err := reference.ParseNamed(dockerName)
	if err != nil {
		return "", errors.Wrap(err, "Error parsing docker registry reference")
	}
	return name.RemoteName(), nil

}

func findOutputRegistry(dockerName string) (string, error) {
	name, err := reference.ParseNamed(dockerName)
	if err != nil {
		return "", errors.Wrap(err, "Error parsing docker registry reference")
	}
	return name.Hostname(), nil
}

func findEnv(vars []api.EnvVar, name string) (string, error) {
	for _, e := range vars {
		if e.Name == name {
			return e.Value, nil
		}
	}
	return "", errors.New("No env variable with name " + name)
}
