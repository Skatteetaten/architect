package config

import (
	"fmt"
	extVersion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/docker"
	"path"
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

func NewBuildInfo(config Config, deliverablePath string) (*BuildInfo, error) {
	buildInfo := BuildInfo{}

	buildInfo.IsSnapshot = isSnapshot(config)

	baseImage, err := createBaseImageInfo(config)

	if err != nil {
		return nil, err
	}

	outputImage, err := createOutputImageInfo(config, deliverablePath)

	if err != nil {
		return nil, err
	}

	//Create completeVersion
	// <application-version>-<builder-version>-<baseimage-repository>-<baseimage-version>
	// e.g. 2.0.0-b1.11.0-oracle8-1.0.2
	completeVersion := outputImage.Version + "-" + config.BuilderSpec.Version + "-" +
		baseImage.Repository + "-" + baseImage.Tags["INFERRED"]

	outputImage.Tags["COMPlETE"] = completeVersion

	buildInfo.BaseImage = *baseImage
	buildInfo.OutputImage = *outputImage

	return &buildInfo, nil
}

func createBaseImageInfo(config Config) (*ImageInfo, error) {
	inferredVersion, err := getBaseImageVersion(config)
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
	versions["LATEST"] = "latest"

	if isSemantic(config) {
		majorVersion, err := getMajor(imageVersion)
		if err != nil {
			return nil, err
		}
		versions["MAJOR"] = majorVersion

		minorVersion, err := getMinor(imageVersion)
		if err != nil {
			return nil, err
		}
		versions["MINOR"] = minorVersion

		versions["PATCH"] = imageVersion
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
		return "", err
	}

	return fmt.Sprintf("%d", build_version.Segments()[0]), nil
}

func getMinor(version string) (string, error) {
	build_version, err := extVersion.NewVersion(version)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d.%d", build_version.Segments()[0], build_version.Segments()[1]), nil
}

func getCompleteVersion(config Config, version string) (string, error) {
	return "domore", nil
}

func isSnapshot(config Config) bool {
	// sjekke config.MavenGav.Version inneholder SNAPSHOT
	return false
}

func getVersion(config Config, isSnapshot bool, deliverablePath string) string {
	if isSnapshot {
		replacer := strings.NewReplacer(config.MavenGav.ArtifactId, "", "Leveransepakke.zip", "")
		return replacer.Replace(path.Base(deliverablePath))
	}

	return config.MavenGav.Version
}

func isSemantic(config Config) bool {
	return false
}

func GetVersionTags() {

}

func getBaseImageVersion(config Config) (string, error) {
	registry := docker.NewRegistryClient(config.DockerSpec.ExternalDockerRegistry)

	manifest, err := registry.PullManifest(config.DockerSpec.BaseImage, config.DockerSpec.BaseVersion)

	if err != nil {
		return "", errors.Wrap(err, "Failed in getBaseImageVersion "+
			"to pull base image manifest from Docker registry")
	}

	biv, err := docker.GetManifestEnv(*manifest, "BASE_IMAGE_VERSION")

	if err != nil {
		return "", errors.Wrap(err, "Failed in getBaseImageVersion "+
			"to extract version from base image manifest")
	} else if biv == "" {
		return "", errors.Wrap(err, "Failed in getBaseImageVersion "+
			"to extract version from base image manifest")
	}

	return biv, nil
}
