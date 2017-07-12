package config

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/reference"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config/api"
	"github.com/skatteetaten/architect/pkg/docker"
	"io/ioutil"
	"os"
	"strings"
)

type ConfigReader interface {
	ReadConfig() (*Config, error)
	AddRegistryCredentials(config *Config) error
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

func (m *FileConfigReader) AddRegistryCredentials(config *Config) error {
	dockerConfigPath, err := docker.GetDockerConfigPath()

	if err != nil {
		return err
	}

	return addRegistryCredentials(config, dockerConfigPath)
}

func (m *InClusterConfigReader) ReadConfig() (*Config, error) {
	buildConfig := os.Getenv("BUILD")

	if len(buildConfig) == 0 {
		return nil, errors.New("Expected a build config environment variable to be present.")
	}

	return newConfig([]byte(buildConfig))
}

func (m *InClusterConfigReader) AddRegistryCredentials(config *Config) error {
	return addRegistryCredentials(config, "/var/run/secrets/openshift.io/push/.dockercfg")
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

	env := make(map[string]string)
	for _, e := range customStrategy.Env {
		env[e.Name] = e.Value
	}

	gav := MavenGav{}
	if artifactId, err := findEnv(env, "ARTIFACT_ID"); err == nil {
		gav.ArtifactId = artifactId
	} else {
		return nil, err
	}
	if groupId, err := findEnv(env, "GROUP_ID"); err == nil {
		gav.GroupId = groupId
	} else {
		return nil, err
	}
	if version, err := findEnv(env, "VERSION"); err == nil {
		gav.Version = version
	} else {
		return nil, err
	}

	if version, err := findEnv(env, "VERSION"); err == nil {
		gav.Version = version
	} else {
		return nil, err
	}

	dockerSpec := DockerSpec{}

	if baseImage, err := findEnv(env, "DOCKER_BASE_IMAGE"); err == nil {
		dockerSpec.BaseImage = baseImage
	} else if baseImage, err := findEnv(env, "DOCKER_BASE_NAME"); err == nil {
		dockerSpec.BaseImage = baseImage
	} else {
		return nil, err
	}

	if externalRegistry, err := findEnv(env, "BASE_IMAGE_REGISTRY"); err == nil {
		if strings.HasPrefix(externalRegistry, "https://") {
			dockerSpec.ExternalDockerRegistry = externalRegistry
		} else {
			dockerSpec.ExternalDockerRegistry = "https://" + externalRegistry
		}
	} else {
		dockerSpec.ExternalDockerRegistry = "https://docker-registry.aurora.sits.no:5000"
	}

	if baseImageVersion, err := findEnv(env, "DOCKER_BASE_VERSION"); err == nil {
		dockerSpec.BaseVersion = baseImageVersion
	} else {
		return nil, err
	}

	dockerSpec.PushExtraTags = "latest,major,minor,patch"
	if pushExtraTags, err := findEnv(env, "PUSH_EXTRA_TAGS"); err == nil {
		dockerSpec.PushExtraTags = pushExtraTags
	}

	dockerSpec.TagWith = ""
	if temporaryTag, err := findEnv(env, "TAG_WITH"); err == nil {
		dockerSpec.TagWith = temporaryTag
	}

	dockerSpec.RetagWith = ""
	if temporaryTag, err := findEnv(env, "RETAG_WITH"); err == nil {
		dockerSpec.RetagWith = temporaryTag
	}

	dockerSpec.TagOverwrite = false
	if tagOverwrite, err := findEnv(env, "TAG_OVERWRITE"); err == nil {
		if strings.Contains(strings.ToLower(tagOverwrite), "true") {
			dockerSpec.TagOverwrite = true
		}
	}

	builderSpec := BuilderSpec{}

	if builderVersion, present := os.LookupEnv("APP_VERSION"); present {
		builderSpec.Version = builderVersion
	} else {
		//We set it to local for local builds.
		//Running on OpenShift will have APP_VERSION as environment variable
		builderSpec.Version = "local"
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

func addRegistryCredentials(cfg *Config, dockerConfigPath string) error {
	_, err := os.Stat(dockerConfigPath)

	if err != nil {
		if os.IsNotExist(err) {
			logrus.Infof("Will not load registry credentials. %s not found.", dockerConfigPath)
			return nil
		}

		return err
	}

	dockerConfigReader, err := os.Open(dockerConfigPath)

	if err != nil {
		return err
	}

	dockerConfig, err := docker.ReadConfig(dockerConfigReader)

	if err != nil {
		return err
	}

	basicCredentials, err := dockerConfig.GetCredentials(cfg.DockerSpec.OutputRegistry)

	if err != nil {
		return err
	} else if basicCredentials == nil {
		logrus.Infof("Will not load registry credentials. No entry for %s in %s.", cfg.DockerSpec.OutputRegistry, dockerConfigPath)
		return nil
	}

	registryCredentials := docker.RegistryCredentials{basicCredentials.User, basicCredentials.Password,
		cfg.DockerSpec.OutputRegistry}

	encodedCredentials, err := registryCredentials.Encode()

	if err != nil {
		return err
	}

	cfg.DockerSpec.OutputRegistryCredentials = encodedCredentials

	return err
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

func findEnv(env map[string]string, name string) (string, error) {
	value, ok := env[name]
	if ok {
		return value, nil
	}
	return "", errors.New("No env variable with name " + name)
}
