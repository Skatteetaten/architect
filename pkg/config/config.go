package config

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/reference"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config/api"
	"io/ioutil"
	"net"
	"os"
	"strings"
)

type ConfigReader interface {
	ReadConfig() (*Config, error)
}

type GAVConfigReader interface {
	ReadGAVConfig() (*MavenGav, error)
}

type InClusterConfigReader struct {
}

type AuroraBuildConfigReader struct {
}

type FileConfigReader struct {
	pathToConfigFile string
}

func NewInClusterConfigReader() ConfigReader {
	return &InClusterConfigReader{}
}

func NewAuroraBuildConfigReader() ConfigReader {
	return &AuroraBuildConfigReader{}
}

func NewFileConfigReader(filepath string) ConfigReader {
	return &FileConfigReader{pathToConfigFile: filepath}
}

func (m *FileConfigReader) ReadConfig() (*Config, error) {
	dat, err := ioutil.ReadFile(m.pathToConfigFile)
	if err != nil {
		return nil, err
	}

	return newConfig(dat, false)
}

func (m *InClusterConfigReader) ReadConfig() (*Config, error) {
	buildConfig := os.Getenv("BUILD")

	if len(buildConfig) == 0 {
		return nil, errors.New("Expected a build config environment variable to be present.")
	}

	return newConfig([]byte(buildConfig), true)
}

//TODO: Create new ReadConfig method when we have a CRD in place

func (m *AuroraBuildConfigReader) ReadConfig() (*Config, error) {

	//ApplicationType
	var applicationType ApplicationType = JavaLeveransepakke
	if appType, exists := os.LookupEnv("APPLICATION_TYPE"); exists {
		if strings.ToUpper(appType) == "NODEJS" {
			applicationType = NodeJsLeveransepakke
		}
	}

	//Application spec
	applicationSpec := ApplicationSpec{}
	mavenGav, err := loadGAVConfig()

	if err != nil {
		return nil, errors.Wrap(err, "Unable to load GAV parameters")
	}
	applicationSpec.MavenGav = mavenGav

	if value, exists := os.LookupEnv("DOCKER_BASE_IMAGE"); exists {
		logrus.Debugf("DOCKER_BASE_IMAGE %s", value)
		applicationSpec.BaseImageSpec.BaseImage = value
	} else {
		logrus.Debug("Fant ikke base image")
		return nil, errors.New("DOCKER_BASE_IMAGE is missing")
	}

	if value, exists := os.LookupEnv("DOCKER_BASE_VERSION"); exists {
		logrus.Debugf("DOCKER_BASE_VERSION %s", value)
		applicationSpec.BaseImageSpec.BaseVersion = value
	} else {
		return nil, errors.New("DOCKER_BASE_VERSION is missing")
	}

	//Docker spec
	dockerSpec := DockerSpec{}

	if value, exists := os.LookupEnv("BASE_IMAGE_REGISTRY"); exists {
		dockerSpec.ExternalDockerRegistry = value
	} else {
		return nil, errors.New("BASE_IMAGE_REGISTRY is missing")
	}

	if value, exists := os.LookupEnv("OUTPUT_REGISTRY"); exists {
		dockerSpec.OutputRegistry = value
	} else {
		return nil, errors.New("OUTPUT_REGISTRY is missing")
	}

	if value, exists := os.LookupEnv("OUTPUT_REPOSITORY"); exists {
		dockerSpec.OutputRepository = value
	} else {
		return nil, errors.New("OUTPUT_REPOSITORY is missing")
	}

	if value, exists := os.LookupEnv("PUSH_EXTRA_TAGS"); exists {
		dockerSpec.PushExtraTags = ParseExtraTags(value)
	} else {
		dockerSpec.PushExtraTags = ParseExtraTags("latest,major,minor,patch")
	}

	if value, exists := os.LookupEnv("TAG_WITH"); exists {
		dockerSpec.TagWith = value
	}

	if value, exists := os.LookupEnv("RETAG_WITH"); exists {
		dockerSpec.RetagWith = value
	}

	dockerSpec.TagOverwrite = false
	if value, exists := os.LookupEnv("TAG_OVERWRITE"); exists {
		if strings.Contains(strings.ToLower(value), "true") {
			dockerSpec.TagOverwrite = true
		}
	}

	//Builder spec
	builderSpec := BuilderSpec{}
	if builderVersion, present := os.LookupEnv("APP_VERSION"); present {
		builderSpec.Version = builderVersion
	} else {
		//We set it to local for local builds.
		//Running on OpenShift will have APP_VERSION as environment variable
		builderSpec.Version = "local"
	}

	c := &Config{
		ApplicationType: applicationType,
		ApplicationSpec: applicationSpec,
		DockerSpec:      dockerSpec,
		BuilderSpec:     builderSpec,
		BinaryBuild:     false, //TODO: Find a way to support binary builds
	}
	return c, nil
}

func loadGAVConfig() (MavenGav, error) {
	GAV := MavenGav{}

	if value, exists := os.LookupEnv("APPLICATION_TYPE"); exists && strings.ToUpper(value) == "NODEJS" {
		GAV.Classifier = Classifier(Webleveransepakke)
		GAV.Type = TgzPackaging
	} else {
		GAV.Classifier = Classifier(Leveransepakke)
		GAV.Type = ZipPackaging
	}

	if value, exists := os.LookupEnv("ARTIFACT_ID"); exists {
		GAV.ArtifactId = value
	} else {
		return MavenGav{}, errors.New("ARTIFACT_ID is missing")
	}

	if value, exists := os.LookupEnv("GROUP_ID"); exists {
		GAV.GroupId = value
	} else {
		return MavenGav{}, errors.New("GROUP_ID is missing")
	}

	if value, exists := os.LookupEnv("VERSION"); exists {
		GAV.Version = value
	} else {
		return MavenGav{}, errors.New("VERSION is missing")
	}

	return GAV, nil
}

func newConfig(buildConfig []byte, rewriteDockerRepositoryName bool) (*Config, error) {
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

	applicationSpec := ApplicationSpec{}
	if artifactId, err := findEnv(env, "ARTIFACT_ID"); err == nil {
		applicationSpec.MavenGav.ArtifactId = artifactId
	} else {
		return nil, err
	}
	if groupId, err := findEnv(env, "GROUP_ID"); err == nil {
		applicationSpec.MavenGav.GroupId = groupId
	} else {
		return nil, err
	}
	if version, err := findEnv(env, "VERSION"); err == nil {
		applicationSpec.MavenGav.Version = version
	} else {
		return nil, err
	}
	if classifier, err := findEnv(env, "CLASSIFIER"); err == nil {
		applicationSpec.MavenGav.Classifier = Classifier(classifier)
	} else {
		if applicationType == JavaLeveransepakke {
			applicationSpec.MavenGav.Classifier = Leveransepakke
		} else {
			applicationSpec.MavenGav.Classifier = Webleveransepakke
		}
	}
	if applicationType == JavaLeveransepakke {
		applicationSpec.MavenGav.Type = ZipPackaging
	} else {
		applicationSpec.MavenGav.Type = TgzPackaging
	}

	if baseSpec, err := findBaseImage(env); err == nil {
		applicationSpec.BaseImageSpec = baseSpec
	} else {
		return nil, err
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

	if temporaryTag, err := findEnv(env, "TAG_WITH"); err == nil {
		dockerSpec.TagWith = temporaryTag
	}

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
	logrus.Debugf("Output Kind is: %s ", outputKind)
	if outputKind == "DockerImage" {
		output := build.Spec.Output.To.Name

		outputRegistry, err := findOutputRegistry(output)
		if err != nil {
			return nil, err
		}
		dockerSpec.OutputRegistry, err = resolveIpIfInternalRegistry(outputRegistry, rewriteDockerRepositoryName)
		if err != nil {
			return nil, err
		}
		dockerSpec.OutputRepository, err = findOutputRepository(output)
		if err != nil {
			return nil, err
		}
		// TAG_WITH environment variable have precedence over tag in output
		if dockerSpec.TagWith == "" {
			dockerSpec.TagWith, err = findOutputTag(output)
			if err != nil {
				dockerSpec.TagWith = ""
			}
		}
	} else if outputKind == "ImageStreamTag" {
		outputRegistry, exists := os.LookupEnv("OUTPUT_REGISTRY")
		if !exists {
			logrus.Error("Expected OUTPUT_REGISTRY environment variable when outputKind is ImageStreamTag")
			return nil, errors.New("No output registry")
		}
		dockerSpec.OutputRegistry, err = resolveIpIfInternalRegistry(outputRegistry, rewriteDockerRepositoryName)
		if err != nil {
			return nil, err
		}
		outputImage, exists := os.LookupEnv("OUTPUT_IMAGE")
		if !exists {
			logrus.Error("Expected OUTPUT_IMAGE environment variable when outputKind is ImageStreamTag")
			return nil, errors.New("No output image")
		}
		dockerUrl := dockerSpec.OutputRegistry + "/" + outputImage
		dockerSpec.TagWith, err = findOutputTag(dockerUrl)
		if err != nil {
			return nil, err
		}
		dockerSpec.OutputRepository, err = findOutputRepository(dockerUrl)
		if err != nil {
			return nil, err
		}
		dockerSpec.PushExtraTags = ParseExtraTags("")
	} else {
		return nil, errors.Errorf("Unknown outputkind. Only DockerImage and ImageStreamTag supported, was %s", outputKind)
	}
	logrus.Debugf("Pushing to %s/%s:%s", dockerSpec.OutputRegistry, dockerSpec.OutputRepository, dockerSpec.TagWith)
	c := &Config{
		ApplicationType: applicationType,
		ApplicationSpec: applicationSpec,
		DockerSpec:      dockerSpec,
		BuilderSpec:     builderSpec,
		BinaryBuild:     build.Spec.Source.Type == api.BuildSourceBinary,
	}
	return c, nil
}

//resolveIpIfInternalRegistry To fix AOT-263
func resolveIpIfInternalRegistry(registryWithPort string, rewrite bool) (string, error) {
	if !rewrite {
		return registryWithPort, nil
	}
	host, port, err := net.SplitHostPort(registryWithPort)
	if err != nil {
		logrus.Warnf("Error splitting host and port of %s.. Try to continue", registryWithPort)
	}
	//We have a IP (OCP < 3.9)
	if net.ParseIP(host) != nil {
		return registryWithPort, nil
	}
	canonical, err := net.LookupCNAME(host)
	if err != nil {
		return "", err
	}
	/*
		We assume that intern registry has cluster.local suffix.. This is valid today, but not necessary in the future
	*/
	if strings.HasSuffix(canonical, "cluster.local.") {
		ips, err := net.LookupIP(canonical)
		if err != nil {
			return "", err
		}
		for _, ip := range ips {
			ipv4 := ip.To4()
			if ipv4 != nil {
				return net.JoinHostPort(ipv4.String(), port), nil
			}
		}
		return "", errors.Errorf("Unable to lookup IP for %s", canonical)
	}
	return registryWithPort, nil
}

func findBaseImage(env map[string]string) (DockerBaseImageSpec, error) {
	baseSpec := DockerBaseImageSpec{}
	if baseImage, err := findEnv(env, "DOCKER_BASE_IMAGE"); err == nil {
		baseSpec.BaseImage = baseImage
	} else if baseImage, err := findEnv(env, "DOCKER_BASE_NAME"); err == nil {
		baseSpec.BaseImage = baseImage
	} else {
		return baseSpec, err
	}

	if baseImageVersion, err := findEnv(env, "DOCKER_BASE_VERSION"); err == nil {
		baseSpec.BaseVersion = baseImageVersion
	} else {
		return baseSpec, err
	}
	return baseSpec, nil
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

func findOutputTag(dockerName string) (string, error) {
	name, err := reference.ParseNamed(dockerName)
	if err != nil {
		return "", errors.Wrap(err, "Error parsing docker registry reference")
	}
	if tagged, isTagged := name.(reference.NamedTagged); isTagged {
		return tagged.Tag(), nil
	}
	return "", errors.Errorf("Could not parse tag from %s", dockerName)
}

func findEnv(env map[string]string, name string) (string, error) {
	value, ok := env[name]
	if ok {
		return value, nil
	}
	return "", errors.New("No env variable with name " + name)
}
