package config

import (
	"fmt"
	extVersion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/docker"
	"path"
	"regexp"
	"strings"
)

/*
EKSEMPEL:
Gitt følgende URL http://uil0map-paas-app01.skead.no:9090/v2/aurora/console/tags/list
=> OutputImage Map[..]
COMPlETE	 		= 2.0.0-b1.11.0-oracle8-1.0.2
LATEST				= latest
MAJOR				= 2
MINOR				= 2.0
PATCH				= 2.0.0
=> OutputImage.Repository	aurora/console
=> BaseImage Map[..]
CONFIG			= 1
INFERRED		= 1.0.2
=> BaseImage.Repository		aurora/oracle8
*/

type BuildInfo struct {
	IsSnapshot  bool
	Version     string
	OutputImage ImageInfo
	BaseImage   ImageInfo
}

type ImageInfo struct {
	Repository string
	Version    string
	Tags       map[string]string
}

func NewBuildInfo(provider docker.ManifestProvider, config Config, deliverable Deliverable) (*BuildInfo, error) {
	buildInfo := BuildInfo{}

	buildInfo.IsSnapshot = isSnapshot(config)

	baseImage, err := createBaseImageInfo(provider, config)

	if err != nil {
		return nil, err
	}

	outputImage, err := createOutputImageInfo(config, deliverable.Path)

	if err != nil {
		return nil, err
	}

	//Create completeVersion
	// <application-version>-<builder-version>-<baseimage-repository>-<baseimage-version>
	// e.g. 2.0.0-b1.11.0-oracle8-1.0.2
	lastNameInRepo := getLastIndexInRepository(baseImage.Repository)

	completeVersion := outputImage.Version
	completeVersion = completeVersion + "-b" + config.BuilderSpec.Version
	completeVersion = completeVersion + "-" + lastNameInRepo
	completeVersion = completeVersion + "-" + baseImage.Tags["INFERRED"]

	outputImage.Tags["COMPlETE"] = completeVersion

	buildInfo.BaseImage = *baseImage
	buildInfo.OutputImage = *outputImage

	return &buildInfo, nil
}

func GetVersionTags(buildInfo BuildInfo) []string {
	versions := buildInfo.OutputImage.Tags
	tags := make([]string, 0, len(versions))
	for _, value := range versions {
		tags = append(tags, value)
	}
	return tags
}

func createBaseImageInfo(provider docker.ManifestProvider, config Config) (*ImageInfo, error) {
	inferredVersion, err := getBaseImageVersion(provider, config)
	if err != nil {
		return nil, errors.Wrap(err, "Error calling getBaseImageVersion in createBaseImageInfo.")
	}
	configVersion := config.DockerSpec.BaseVersion
	respositoryName := config.DockerSpec.BaseImage

	versions := map[string]string{
		"CONFIG":   configVersion,
		"INFERRED": inferredVersion,
	}

	return &ImageInfo{respositoryName, configVersion, versions}, nil
}

func createOutputImageInfo(config Config, deliverablePath string) (*ImageInfo, error) {
	b := isSnapshot(config)

	imageVersion := getVersion(config, b, deliverablePath)

	versions := make(map[string]string)

	if strings.Contains(config.DockerSpec.PushExtraTags, "latest") {
		versions["LATEST"] = "latest"
	}

	if isSemantic(config) {
		if strings.Contains(config.DockerSpec.PushExtraTags, "major") {
			majorVersion, err := getMajor(imageVersion)
			if err != nil {
				return nil, err
			}

			versions["MAJOR"] = majorVersion
		}

		if strings.Contains(config.DockerSpec.PushExtraTags, "minor") {
			minorVersion, err := getMinor(imageVersion)
			if err != nil {
				return nil, err
			}
			versions["MINOR"] = minorVersion
		}

		if strings.Contains(config.DockerSpec.PushExtraTags, "patch") {
			versions["PATCH"] = imageVersion
		}
	}

	/*versions := map[string]string{
		"COMPlETE_VERSION": "2.0.0-b1.11.0-oracle8-1.0.2",
		"LATEST":           "latest",
		"MAJOR":            "2",
		"MINOR":            "2.0",
		"PATCH":            "2.0.0",
	}*/

	return &ImageInfo{config.DockerSpec.OutputRepository, imageVersion, versions}, nil
}

func getMajor(version string) (string, error) {
	build_version, err := extVersion.NewVersion(version)

	if err != nil {
		return "", errors.Wrap(err, "Error in parsing major version: "+version)
	}

	return fmt.Sprintf("%d", build_version.Segments()[0]), nil
}

func getMinor(version string) (string, error) {
	build_version, err := extVersion.NewVersion(version)

	if err != nil {
		return "", errors.Wrap(err, "Error in parsing minor version: "+version)
	}

	return fmt.Sprintf("%d.%d", build_version.Segments()[0], build_version.Segments()[1]), nil
}

func getLastIndexInRepository(repository string) string {
	s := strings.Split(repository, "/")
	return s[len(s)-1]
}

func isSnapshot(config Config) bool {
	if strings.Contains(config.MavenGav.Version, "SNAPSHOT") {
		return true
	}
	return false
}

func isSemantic(config Config) bool {
	var validStr = regexp.MustCompile(`^[0-9]+.[0-9]+.[0-9]+$`)
	if validStr.MatchString(config.MavenGav.Version) {
		return true
	}
	return false
}

func getVersion(config Config, isSnapshot bool, deliverablePath string) string {
	if isSnapshot {
		replacer := strings.NewReplacer(config.MavenGav.ArtifactId, "", "-Leveransepakke.zip", "")
		return "SNAPSHOT" + replacer.Replace(path.Base(deliverablePath))
	}

	return config.MavenGav.Version
}

func getBaseImageVersion(provider docker.ManifestProvider, config Config) (string, error) {
	biv, err := provider.GetManifestEnv(config.DockerSpec.BaseImage, config.DockerSpec.BaseVersion, "BASE_IMAGE_VERSION")

	if err != nil {
		return "", err
	} else if biv == "" {
		return "", fmt.Errorf("Failed to extract version in getBaseImageVersion, registry: %s, "+
			"BaseImage: %s, BaseVersion: %s ",
			provider, config.DockerSpec.BaseImage, config.DockerSpec.BaseVersion)
	}
	return biv, nil
}
