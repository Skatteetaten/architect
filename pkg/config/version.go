package config

import (
	extVersion "github.com/hashicorp/go-version"
	"fmt"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/pkg/errors"
)
/*
EKSEMPEL:
Gitt følgende URL http://uil0map-paas-app01.skead.no:9090/v2/aurora/console/tags/list
=> OutputImage Map[..]
COMPlETE_VERSION 		= 2.0.0-b1.11.0-oracle8-1.0.2
LATEST				= latest
MAJOR				= 2
MINOR				= 2.0
PATCH				= 2.0.0
=> OutputImage.Repository	aurora/console
=> BaseImage Map[..]
CONFIG_VERSION			= 1
INFERRED_VERSION		= 1.0.2
=> BaseImage.Repository		aurora/oracle8
 */

type BuildInfo struct {
	IsSnapshot		bool
	OutputImage		ImageInfo
	BaseImage		ImageInfo
}

type ImageInfo struct {
	Repository	string
	Version 	string
	Tags		map[string]string
}


func NewBuildInfo(config Config, filepath string) (*BuildInfo, error) {
	buildInfo := BuildInfo{}

	buildInfo.IsSnapshot = isSnapshot(config)

	baseImage, err := createBaseImageInfo(config)

	if err != nil {
		return nil, err
	}

	outputImage, err := createOutputImageInfo(config)

	if err != nil {
		return nil, err
	}

	//Create completeVersion
	// <application-version>-<builder-version>-<baseimage-repository>-<baseimage-version>
	// e.g. 2.0.0-b1.11.0-oracle8-1.0.2
	completeVersion := outputImage.Version + "-" + config.BuilderSpec.Version + "-" + baseImage.Repository + "-" + baseImage.Tags["INFERRED_VERSION"]

	outputImage.Tags["COMPlETE_VERSION"] = completeVersion

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
		"CONFIG_VERSION": 	configVersion,
		"INFERRED_VERSION":     inferredVersion,
	}

	return &ImageInfo{respositoryName, configVersion, versions }, nil
}

func createOutputImageInfo(config Config) (*ImageInfo, error) {
	imageVersion, err := getVersion(config)
	if err != nil {
		return nil, err
	}

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

	return &ImageInfo{config.DockerSpec.OutputRepository, imageVersion, versions }, nil
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

	return fmt.Sprintf("%d.%d", build_version.Segments()[0],build_version.Segments()[1]), nil
}

func getCompleteVersion(config Config, version string) (string, error) {
	return "domore", nil
}

func isSnapshot(config Config) (bool) {
	// sjekke config.MavenGav.Version inneholder SNAPSHOT
	return false
}

func isSemantic(config Config) (bool) {
	return false
}

func getVersion(config Config) (string, error) {
	//re, _ := regexp.Compile(".*SNAPSHOT.*")
	/*
	if [[ $VERSION =~ "SNAPSHOT"  ]]; then
  SNAPSHOT_TAG=$VERSION
  versionSuffix=$(echo $zipFile | sed "s/$ARTIFACT_ID-//g;s/-Leveransepakke.zip//g")
  VERSION="SNAPSHOT-$versionSuffix"
fi
	 */
	return "domore", nil
}

func GetVersionTags() () {

}

func getBaseImageVersion(config Config) (string, error) {
	registry := docker.NewRegistryClient(config.DockerSpec.ExternalDockerRegistry)

	manifest, err := registry.PullManifest(config.DockerSpec.BaseImage, config.DockerSpec.BaseVersion)

	if err != nil {
		return "", errors.Wrap(err,"Failed in getBaseImageVersion " +
			"to pull base image manifest from Docker registry")
	}

	biv, err := docker.GetManifestEnv(*manifest, "BASE_IMAGE_VERSION")

	if err != nil {
		return "", errors.Wrap(err, "Failed in getBaseImageVersion " +
			"to extract version from base image manifest")
	} else if biv == "" {
		return "", errors.Wrap(err,"Failed in getBaseImageVersion " +
			"to extract version from base image manifest")
	}

	return biv, nil
}