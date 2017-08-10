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

	env := make(map[string]string)
	for _, e := range customStrategy.Env {
		env[e.Name] = e.Value
	}

	var applicationType ApplicationType = JavaLeveransepakke
	if appType, err := findEnv(env, "APPLICATION_TYPE"); err == nil {
		if strings.ToUpper(appType) == "NODEJS" {
			applicationType = NodeJsLeveransepakke
		}
	}

	var javaApp *JavaApplication = nil
	var nodeApp *NodeApplication = nil
	if applicationType == JavaLeveransepakke {
		javaApp = &JavaApplication{}
		if artifactId, err := findEnv(env, "ARTIFACT_ID"); err == nil {
			javaApp.ArtifactId = artifactId
		} else {
			return nil, err
		}
		if groupId, err := findEnv(env, "GROUP_ID"); err == nil {
			javaApp.GroupId = groupId
		} else {
			return nil, err
		}
		if version, err := findEnv(env, "VERSION"); err == nil {
			javaApp.Version = version
		} else {
			return nil, err
		}
		if classifier, err := findEnv(env, "CLASSIFIER"); err == nil {
			javaApp.Classifier = classifier
		} else {
			javaApp.Classifier = "Leveransepakke"
		}
		baseSpec := DockerBaseImageSpec{}
		if baseImage, err := findEnv(env, "DOCKER_BASE_IMAGE"); err == nil {
			baseSpec.BaseImage = baseImage
		} else if baseImage, err := findEnv(env, "DOCKER_BASE_NAME"); err == nil {
			baseSpec.BaseImage = baseImage
		} else {
			return nil, err
		}

		if baseImageVersion, err := findEnv(env, "DOCKER_BASE_VERSION"); err == nil {
			baseSpec.BaseVersion = baseImageVersion
		} else {
			return nil, err
		}
		javaApp.BaseImageSpec = baseSpec

	} else {
		nodeApp = &NodeApplication{}
		if groupId, err := findEnv(env, "NPM_NAME"); err == nil {
			nodeApp.NpmName = groupId
		} else {
			return nil, err
		}
		if v, err := findEnv(env, "VERSION"); err == nil {
			nodeApp.Version = v
		} else {
			return nil, err
		}
		nodeApp.NginxBaseImageSpec = DockerBaseImageSpec{
			BaseImage:   "aurora/modem",
			BaseVersion: "latest",
		}
		nodeApp.NodejsBaseImageSpec = DockerBaseImageSpec{
			BaseImage:   "aurora/wrench",
			BaseVersion: "latest",
		}
	}

	dockerSpec := DockerSpec{}

	if externalRegistry, err := findEnv(env, "BASE_IMAGE_REGISTRY"); err == nil {
		if strings.HasPrefix(externalRegistry, "https://") {
			dockerSpec.ExternalDockerRegistry = externalRegistry
		} else {
			dockerSpec.ExternalDockerRegistry = "https://" + externalRegistry
		}
	} else {
		dockerSpec.ExternalDockerRegistry = "https://docker-registry.aurora.sits.no:5000"
	}

	if pushExtraTags, err := findEnv(env, "PUSH_EXTRA_TAGS"); err == nil {
		dockerSpec.PushExtraTags = ParseExtraTags(pushExtraTags)
	} else {
		dockerSpec.PushExtraTags = ParseExtraTags("latest,major,minor,patch")
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
		ApplicationType:   applicationType,
		JavaApplication:   javaApp,
		NodeJsApplication: nodeApp,
		DockerSpec:        dockerSpec,
		BuilderSpec:       builderSpec,
		BinaryBuild:       build.Spec.Source.Type == api.BuildSourceBinary,
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

func findEnv(env map[string]string, name string) (string, error) {
	value, ok := env[name]
	if ok {
		return value, nil
	}
	return "", errors.New("No env variable with name " + name)
}
