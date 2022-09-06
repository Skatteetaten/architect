package config

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/docker/distribution/reference"
	buildv1 "github.com/openshift/api/build/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Reader interface that provides the ReadConfig method
type Reader interface {
	ReadConfig() (*Config, error)
}

// InClusterConfigReader reads build configuration from the builder pod
type InClusterConfigReader struct {
}

// FileConfigReader reads build configuration from file
type FileConfigReader struct {
	pathToConfigFile string
}

// CmdConfigReader reads build configuration from commandline parameters
type CmdConfigReader struct {
	NoPush bool
	Cmd    *cobra.Command
	Args   []string
}

// NewInClusterConfigReader returns a Reader of type InClusterConfigReader
func NewInClusterConfigReader() Reader {
	return &InClusterConfigReader{}
}

// NewFileConfigReader returns a Reader of type FileConfigReader
func NewFileConfigReader(filepath string) Reader {
	return &FileConfigReader{pathToConfigFile: filepath}
}

// NewCmdConfigReader returns a Reader of type CmdConfigReader
func NewCmdConfigReader(cmd *cobra.Command, args []string, noPush bool) Reader {
	return &CmdConfigReader{
		Cmd:    cmd,
		Args:   args,
		NoPush: noPush,
	}
}

// ReadConfig from commandline parameters
func (m *CmdConfigReader) ReadConfig() (*Config, error) {

	var applicationType = JavaLeveransepakke

	if len(m.Cmd.Flag("type").Value.String()) != 0 {
		value := m.Cmd.Flag("type").Value.String()
		if strings.ToLower(value) == "java" {
			applicationType = JavaLeveransepakke
		} else if strings.ToLower(value) == "nodejs" {
			applicationType = NodeJsLeveransepakke
		} else if strings.ToLower(value) == "doozer" {
			applicationType = DoozerLeveranse
		}
	}

	fromraw := m.Cmd.Flag("from").Value.String()
	from := strings.Split(fromraw, ":")
	if len(from) != 2 {
		return nil, errors.New("--from: baseimage is malformed: " + fromraw)
	}

	outputraw := m.Cmd.Flag("output").Value.String()
	output := strings.Split(outputraw, ":")
	if len(output) != 2 {
		return nil, errors.New("--output: repository is malformed: " + outputraw)
	}

	pushRegistry := m.Cmd.Flag("push-registry").Value.String()
	pullRegistry := m.Cmd.Flag("pull-registry").Value.String()

	if !strings.Contains(pullRegistry, "http") {
		pullRegistry = fmt.Sprintf("https://%s", pullRegistry)
	}

	return &Config{
		NoPush:          m.NoPush,
		BinaryBuild:     true,
		LocalBuild:      true,
		ApplicationType: applicationType,
		ApplicationSpec: ApplicationSpec{
			MavenGav: MavenGav{
				Version: output[1],
			},
			BaseImageSpec: DockerBaseImageSpec{
				BaseImage:   from[0],
				BaseVersion: from[1],
			},
		},
		DockerSpec: DockerSpec{
			ExternalDockerRegistry: pushRegistry,
			InternalPullRegistry:   pullRegistry,
			OutputRegistry:         pushRegistry,
			OutputRepository:       output[0],
			TagWith:                output[1],
		},
		BuildTimeout: 900,
	}, nil

}

// ReadNexusConfigFromFileSystem read nexusUrl, nexusUser, and nexusPassword from file and return NexusAccess
func ReadNexusConfigFromFileSystem() (*NexusAccess, error) {
	nexusAccess := NexusAccess{}
	secretPath := "/u01/nexus/nexus.json"
	jsonFile, err := ioutil.ReadFile(secretPath)
	if err == nil {
		var data map[string]interface{}
		err := json.Unmarshal(jsonFile, &data)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not parse %s. Must be correct json when specified.", secretPath)
		}
		nexusAccess.NexusURL = data["nexusUrl"].(string)
		nexusAccess.Username = data["username"].(string)
		nexusAccess.Password = data["password"].(string)
	} else {
		return nil, errors.Errorf("Could not read nexus config at %s, error: %s", secretPath, err)
	}
	return &nexusAccess, nil
}

// ReadNexusAccessFromEnvVars read nexusUrl, nesusUser, and nexusPassword from env variables and return NexusAccess
func ReadNexusAccessFromEnvVars() (*NexusAccess, error) {
	nexusAccess := NexusAccess{}
	nexusAccess.Username, _ = os.LookupEnv("NEXUS_USERNAME")
	nexusAccess.Password, _ = os.LookupEnv("NEXUS_PASSWORD")
	nexusAccess.NexusURL, _ = os.LookupEnv("NEXUS_URL")
	if nexusAccess.IsValid() {
		return &nexusAccess, nil
	}
	return nil, errors.Errorf("Could not read Nexus credentials from environment")
}

// ReadConfig from file
func (m *FileConfigReader) ReadConfig() (*Config, error) {
	dat, err := ioutil.ReadFile(m.pathToConfigFile)
	if err != nil {
		return nil, err
	}

	return newConfig(dat, false)
}

// ReadConfig from the buildConfig
func (m *InClusterConfigReader) ReadConfig() (*Config, error) {
	buildConfig := os.Getenv("BUILD")

	if len(buildConfig) == 0 {
		return nil, errors.New("expected a build config environment variable to be present")
	}

	return newConfig([]byte(buildConfig), true)
}

func newConfig(buildConfig []byte, rewriteDockerRepositoryName bool) (*Config, error) {
	build := buildv1.Build{}
	err := json.Unmarshal(buildConfig, &build)
	if err != nil {
		return nil, err
	}
	customStrategy := build.Spec.Strategy.CustomStrategy
	if customStrategy == nil {
		return nil, errors.New("expected strategy to be custom strategy. thats the only one supported")
	}

	binaryBuild := build.Spec.Source.Type == buildv1.BuildSourceBinary

	env := make(map[string]string)
	for _, e := range customStrategy.Env {
		env[e.Name] = e.Value
	}

	var applicationType = JavaLeveransepakke
	if appType, err := findEnv(env, "APPLICATION_TYPE"); err == nil {
		if strings.ToUpper(appType) == NodeJs {
			applicationType = NodeJsLeveransepakke
		} else if strings.ToUpper(appType) == Doozer {
			applicationType = DoozerLeveranse
		}
	}

	var sporingstjeneste = ""
	if value, err := findEnv(env, "SPORINGSTJENESTE"); err == nil && value != "" {
		logrus.Debugf("Sporingstjeneste: %s", value)
		sporingstjeneste = value
	}

	var tlsVerify = true
	if value, err := findEnv(env, "TLS_VERIFY"); err == nil {
		if strings.Contains(strings.ToLower(value), "false") {
			tlsVerify = false
		}
	}

	var buildTimeout time.Duration = 900
	if value, err := findEnv(env, "BUILD_TIMEOUT_IN_S"); err == nil {
		i, err := strconv.Atoi(value)
		if err == nil {
			buildTimeout = time.Duration(i)
		}
	}

	applicationSpec := ApplicationSpec{}
	if artifactID, err := findEnv(env, "ARTIFACT_ID"); err == nil {
		applicationSpec.MavenGav.ArtifactID = artifactID
	} else {
		return nil, err
	}
	if groupID, err := findEnv(env, "GROUP_ID"); err == nil {
		applicationSpec.MavenGav.GroupID = groupID
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
		} else if applicationType == NodeJsLeveransepakke {
			applicationSpec.MavenGav.Classifier = Webleveransepakke
		} else {
			applicationSpec.MavenGav.Classifier = Doozerleveransepakke
		}
	}
	if applicationType == JavaLeveransepakke || applicationType == DoozerLeveranse {
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

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	if externalRegistry, err := findEnv(env, "BASE_IMAGE_REGISTRY"); err == nil {
		if strings.HasPrefix(externalRegistry, "https://") {
			dockerSpec.ExternalDockerRegistry = externalRegistry
		} else {
			dockerSpec.ExternalDockerRegistry = "https://" + externalRegistry
		}
	} else if strings.ToLower(build.Spec.CommonSpec.Output.To.Kind) == "dockerimage" {
		registryURL, err := url.Parse("https://" + build.Spec.CommonSpec.Output.To.Name)
		if err != nil {
			logrus.Errorf("Failed to parse dockerimage-url from BC for ExternalDockerRegistry")
		} else {
			base := registryURL.Host
			if err := checkURL(client, "https://", base, "/v2/"); err == nil {
				dockerSpec.ExternalDockerRegistry = "https://" + base
				logrus.Debugf("Using https: %s", dockerSpec.ExternalDockerRegistry)
			} else if err := checkURL(client, "http://", base, "/v2/"); err == nil {
				dockerSpec.ExternalDockerRegistry = "http://" + base
				logrus.Debugf("Using insecure registry: %s", dockerSpec.ExternalDockerRegistry)
			} else {
				logrus.Errorf("Failed to access url %s from BC for ExternalDockerRegistry", base)
			}
		}
	} else {
		//If all fails
		logrus.Errorf("Failed to find a specified url for ExternalDockerRegistry")
	}

	if internalPullRegistry, err := findEnv(env, "INTERNAL_PULL_REGISTRY"); err == nil {
		base := internalPullRegistry
		if err := checkURL(client, "https://", base, "/v2/"); err == nil {
			dockerSpec.InternalPullRegistry = "https://" + base
			logrus.Debugf("Using https: %s", dockerSpec.ExternalDockerRegistry)
		} else if err := checkURL(client, "http://", base, "/v2/"); err == nil {
			dockerSpec.InternalPullRegistry = "http://" + base
			logrus.Debugf("Using insecure registry: %s", dockerSpec.ExternalDockerRegistry)
		} else {
			logrus.Errorf("Failed to access url %s for InternalPullRegistry.", internalPullRegistry)
		}
	} else {
		logrus.Error("Failed to find a specified url for InternalPullRegistry")
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

	// TagOverwrite has been removed since 10.2021. Kept to logg usage
	if _, err := findEnv(env, "TAG_OVERWRITE"); err == nil {
		logrus.Warning("Functionality for TAG_OVERWRITE has been removed")
	}

	buildType := Snapshot
	if envBuildType, err := findEnv(env, "BINARY_BUILD_TYPE"); err == nil {
		buildType = BinaryBuildType(envBuildType)
	}

	var nexusIqReportURL string
	if envNexusIqReportUrl, err := findEnv(env, "IMAGE_LABEL_NEXUS_IQ_REPORT_URL"); err == nil {
		nexusIqReportURL = envNexusIqReportUrl
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
		dockerSpec.OutputRegistry, err = resolveIPIfInternalRegistry(outputRegistry, rewriteDockerRepositoryName)
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
		dockerSpec.OutputRegistry, err = resolveIPIfInternalRegistry(outputRegistry, rewriteDockerRepositoryName)
		if err != nil {
			return nil, err
		}
		outputImage, exists := os.LookupEnv("OUTPUT_IMAGE")
		if !exists {
			logrus.Error("Expected OUTPUT_IMAGE environment variable when outputKind is ImageStreamTag")
			return nil, errors.New("No output image")
		}
		dockerURL := dockerSpec.OutputRegistry + "/" + outputImage
		dockerSpec.TagWith, err = findOutputTag(dockerURL)
		if err != nil {
			return nil, err
		}
		dockerSpec.OutputRepository, err = findOutputRepository(dockerURL)
		if err != nil {
			return nil, err
		}
		dockerSpec.PushExtraTags = ParseExtraTags("")
	} else {
		return nil, errors.Errorf("Unknown outputkind. Only DockerImage and ImageStreamTag supported, was %s", outputKind)
	}
	logrus.Debugf("Pushing to %s/%s:%s", dockerSpec.OutputRegistry, dockerSpec.OutputRepository, dockerSpec.TagWith)

	c := &Config{
		ApplicationType:   applicationType,
		ApplicationSpec:   applicationSpec,
		DockerSpec:        dockerSpec,
		BuilderSpec:       builderSpec,
		BinaryBuild:       binaryBuild,
		TLSVerify:         tlsVerify,
		BuildTimeout:      buildTimeout,
		Sporingstjeneste:  sporingstjeneste,
		OwnerReferenceUid: string(build.UID),
		BinaryBuildType:   buildType,
		NexusIQReportUrl:  nexusIqReportURL,
	}
	return c, nil
}

func checkURL(client *http.Client, protocol string, base string, path string) error {
	res, err := client.Get(protocol + base + path)
	if err == nil {
		defer res.Body.Close()
		return nil
	}
	return err
}

// resolveIPIfInternalRegistry To fix AOT-263
func resolveIPIfInternalRegistry(registryWithPort string, rewrite bool) (string, error) {
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

	name, err := reference.ParseNormalizedNamed(dockerName)

	//name, err := reference.ParseNamed(dockerName)
	if err != nil {
		return "", errors.Wrap(err, "Error parsing docker registry reference")
	}

	return reference.Path(name), nil

}

func findOutputRegistry(dockerName string) (string, error) {
	name, err := reference.ParseNamed(dockerName)
	if err != nil {
		return "", errors.Wrap(err, "Error parsing docker registry reference")
	}
	return reference.Domain(name), nil
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
