package config

import (
	extVersion "github.com/hashicorp/go-version"
	"fmt"
	"github.com/skatteetaten/architect/pkg/docker"
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
	Tags		map[string]string
}


func NewBuildInfo(config Config) (BuildInfo, error) {
	buildInfo := BuildInfo{}

	buildInfo.IsSnapshot = isSnapshot(config)

	baseImage, err := createBaseImageInfo(config)

	if err != nil {
		return nil, err
	}
	buildInfo.BaseImage = *baseImage

	outputImage, err := createOutputImageInfo(config)

	if err != nil {
		return nil, err
	}
	buildInfo.OutputImage = *outputImage

	return buildInfo, nil
}

func createBaseImageInfo(config Config) (*ImageInfo, error) {
	versions := map[string]string{
		"CONFIG_VERSION": 	"1",
		"INFERRED_VERSION":     "1.0.2",
	}

	return &ImageInfo{"aurora/oracle8", versions }, nil
}

func createOutputImageInfo(config Config) (*ImageInfo, error) {
	versions := map[string]string{
		"COMPlETE_VERSION": "2.0.0-b1.11.0-oracle8-1.0.2",
		"LATEST":           "latest",
		"MAJOR":            "2",
		"MINOR":            "2.0",
		"PATCH":            "2.0.0",
	}

	return &ImageInfo{"aurora/beastie2", versions }, nil
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

func getBaseImageVersion(config Config, registryAddress string) (string, error) {

	registry := docker.NewRegistryClient(registryAddress)

	manifest, err := registry.PullManifest(config.DockerSpec.BaseImage, config.DockerSpec.BaseVersion)

	if err != nil {
		return "", fmt.Errorf("Failed to pull base image manifest from Docker registry")
	}

	biv, err := docker.GetManifestEnv(*manifest, "BASE_IMAGE_VERSION")

	if err != nil {
		return "", fmt.Errorf("Failed to extract version from base image manifest: %v", err)
	} else if biv == "" {
		return "", fmt.Errorf("Failed to extract version from base image manifest")
	}

	return biv, nil
}