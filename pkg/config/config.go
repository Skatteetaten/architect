package config

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/reference"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config/api"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
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

	return newConfig(dat, false)
}

func (m *InClusterConfigReader) ReadConfig() (*Config, error) {
	buildConfig := os.Getenv("BUILD")

	if len(buildConfig) == 0 {
		return nil, errors.New("Expected a build config environment variable to be present.")
	}

	return newConfig([]byte(buildConfig), true)
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

	var buildStrategy = Docker
	if value, err := findEnv(env, "BUILD_STRATEGY"); err == nil {
		if strings.Contains(strings.ToLower(value), Buildah) {
			buildStrategy = Buildah
		} else {
			buildStrategy = value
		}
	}

	var tlsVerify = true
	if value, err := findEnv(env, "TLS_VERIFY"); err == nil {
		if strings.Contains(strings.ToLower(value), "false") {
			tlsVerify = false
		}
	}

	nexusAccess := NexusAccess{
		NexusUrl: "https://aurora/nexus/service/local/artifact/maven/content",
	}
	secretMountPath := ""
	for _, secret := range customStrategy.Secrets {
		logrus.Debugf("Found secret %s", secret.SecretSource.Name)
		if secret.SecretSource.Name == "jenkins-slave-nexus" {
			secretMountPath = secret.MountPath
		}
	}
	if secretMountPath != "" {
		var secretPath = secretMountPath + "/nexus.json"
		jsonFile, err := ioutil.ReadFile(secretPath)
		if err == nil {
			var data map[string]interface{}
			err := json.Unmarshal(jsonFile, &data)
			if err != nil {
				return nil, errors.Wrapf(err, "Could not parse %s. Must be correct json when specified.", secretPath)
			}
			nexusAccess.NexusUrl = data["nexusUrl"].(string)
			nexusAccess.Username = data["username"].(string)
			nexusAccess.Password = data["password"].(string)
		} else {
			logrus.Warnf("Could not read nexus config at %s, error: %s", secretPath, err)
		}
	} else {
		logrus.Debugf("Found no nexus secret")
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
	} else if strings.ToLower(build.Spec.CommonSpec.Output.To.Kind) == "dockerimage" {
		registryUrl, err := url.Parse("https://" + build.Spec.CommonSpec.Output.To.Name)
		if err != nil {
			dockerSpec.ExternalDockerRegistry = "https://docker-registry.aurora.sits.no:5000"
		} else {
			base := registryUrl.Host
			if _, err := http.Get("https://" + base); err == nil {
				dockerSpec.ExternalDockerRegistry = "https://" + base
				logrus.Debugf("Using https: %s", dockerSpec.ExternalDockerRegistry)
			} else if _, err := http.Get("http://" + base); err == nil {
				dockerSpec.ExternalDockerRegistry = "http://" + base
				logrus.Debugf("Using insecure registry: %s", dockerSpec.ExternalDockerRegistry)
			} else {
				dockerSpec.ExternalDockerRegistry = "https://docker-registry.aurora.sits.no:5000"
			}
		}

	} else {
		//If all fails
		dockerSpec.ExternalDockerRegistry = "https://docker-registry.aurora.sits.no:5000"
	}

	if internalPullRegistry, err := findEnv(env, "INTERNAL_PULL_REGISTRY"); err == nil {
		base := internalPullRegistry
		if _, err := http.Get("https://" + base); err == nil {
			dockerSpec.InternalPullRegistry = "https://" + base
			logrus.Debugf("Using https: %s", dockerSpec.ExternalDockerRegistry)
		} else if _, err := http.Get("http://" + base); err == nil {
			dockerSpec.InternalPullRegistry = "http://" + base
			logrus.Debugf("Using insecure registry: %s", dockerSpec.ExternalDockerRegistry)
		} else {
			dockerSpec.InternalPullRegistry = "https://docker-registry.aurora.sits.no:5000"
		}
	} else {
		dockerSpec.InternalPullRegistry = "https://docker-registry.aurora.sits.no:5000"
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
		NexusAccess:     nexusAccess,
		BinaryBuild:     build.Spec.Source.Type == api.BuildSourceBinary,
		BuildStrategy:   buildStrategy,
		TlsVerify:       tlsVerify,
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
