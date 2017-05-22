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
COMPlETE	 		= 2.0.0-b1.11.0-oracle8-1.0.2
LATEST				= latest
MAJOR				= 2
MINOR				= 2.0
PATCH				= 2.0.0
OutputImage.Repository	aurora/console

*/

type TagInfo struct {
	VersionTags	[]string
}

type BuildInfo struct {
	Env 		map[string]string
	OutputImage	OutputImageInfo
	BaseImage   	BaseImageInfo
}

type OutputImageInfo struct {
	Repository	string
	TagInfo
}

type BaseImageInfo struct {
	Repository	string
	Version		string
}

func NewTagInfo(appVersion string, auroraVersion string, extraTags string) (*TagInfo, error) {
	versionTags, err := getVersionTags(appVersion, auroraVersion, extraTags)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize builder")
	}

	return &TagInfo{VersionTags:versionTags}, nil
}

func NewBuildInfo(cfg Config, deliverable Deliverable, provider docker.ManifestProvider) (*BuildInfo, error) {

	baseImageVersion, err := getBaseImageVersion(provider, cfg)

	if err != nil {
		return nil, errors.Wrap(err, "Error calling getBaseImageVersion in NewBuildInfo.")
	}

	appVersion := getAppVersion(cfg, deliverable.Path)
	auroraVersion := getAuroraVersion(baseImageVersion, appVersion, cfg)

	baseImage := createBaseImageInfo(baseImageVersion, cfg)

	outputImage, err := createOutputImageInfo(appVersion, auroraVersion, cfg)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to get information about output image")
	}

	env := createEnv(appVersion, auroraVersion, cfg)

	return &BuildInfo{Env: env, OutputImage: *outputImage, BaseImage: *baseImage}, nil
}

func createOutputImageInfo(appVersion string, auroraVersion string, cfg Config) (*OutputImageInfo, error) {
	var tags []string
	var err error

	if ! isTemporary(cfg) {
		tags, err = getVersionTags(auroraVersion, appVersion, cfg.DockerSpec.PushExtraTags)

		if err != nil {
			return nil, errors.Wrap(err, "Failed to get tags")
		}
	} else {
		tags = getTemporaryTags(cfg.DockerSpec.TagWith)
	}

	return &OutputImageInfo{Repository: cfg.DockerSpec.OutputRepository, TagInfo: TagInfo{VersionTags: tags}}, nil
}

func createBaseImageInfo(version string, cfg Config) *BaseImageInfo {
	return &BaseImageInfo{Repository: cfg.DockerSpec.BaseImage, Version: version}
}

func createEnv(appVersion string, auroraVersion string, cfg Config) map[string]string {
	env := make(map[string]string)

	env[docker.ENV_APP_VERSION] 	= appVersion
	env[docker.ENV_AURORA_VERSION] 	= auroraVersion
	env[docker.ENV_PUSH_EXTRA_TAGS] = cfg.DockerSpec.PushExtraTags

	if isSnapshot(cfg) {
		env[docker.ENV_SNAPSHOT_TAG] = cfg.MavenGav.Version
	}

	return env
}

func getTemporaryTags(tempVersion string) []string {
	return []string{tempVersion}
}

func getVersionTags(appVersion string, auroraVersion string, extraTags string) ([]string, error) {
	versions := make([]string, 0, 10)

	if strings.Contains(extraTags, "latest") {
		versions = append(versions, "latest")
	}

	versions = append(versions, auroraVersion)

	if isSemantic(appVersion) {
		if strings.Contains(extraTags, "major") {
			majorVersion, err := getMajor(appVersion)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get major version")
			}

			versions = append(versions, majorVersion)
		}

		if strings.Contains(extraTags, "minor") {
			minorVersion, err := getMinor(appVersion)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get minor version")
			}
			versions = append(versions, minorVersion)
		}

		if strings.Contains(extraTags, "patch") {
			versions = append(versions, appVersion)
		}
	}

	return versions, nil
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

func isSnapshot(config Config) bool {
	if strings.Contains(config.MavenGav.Version, "SNAPSHOT") {
		return true
	}
	return false
}

func isSemantic(version string) bool {
	var validStr = regexp.MustCompile(`^[0-9]+.[0-9]+.[0-9]+$`)
	if validStr.MatchString(version) {
		return true
	}
	return false
}

func isTemporary(config Config) bool {
	return config.DockerSpec.TagWith != ""
}

func getBaseImageVersion(provider docker.ManifestProvider, config Config) (string, error) {
	biv, err := provider.GetManifestEnv(config.DockerSpec.BaseImage, config.DockerSpec.BaseVersion, "BASE_IMAGE_VERSION")

	if err != nil {
		return "", errors.Wrap(err, "Failed to extract version in getBaseImageVersion")
	} else if biv == "" {
		return "", errors.Errorf("Failed to extract version in getBaseImageVersion, registry: %s, "+
			"BaseImage: %s, BaseVersion: %s ",
			provider, config.DockerSpec.BaseImage, config.DockerSpec.BaseVersion)
	}
	return biv, nil
}

/*
  Create aurora version aka complete version
  <application-version>-<builder-version>-<baseimage-repository>-<baseimage-version>
  e.g. 2.0.0-b1.11.0-oracle8-1.0.2
 */
func getAuroraVersion(baseImageVersion, appVersion string, cfg Config) string {
	builderVersion := cfg.BuilderSpec.Version
	lastNameInRepo := getLastIndexInRepository(cfg.DockerSpec.BaseImage)

	return fmt.Sprintf("%s-b%s-%s-%s", appVersion, builderVersion, lastNameInRepo, baseImageVersion)
}

/*
  Create app version. If not snapshot build, then return version from GAV.
  Otherwise, create new snapshot version based on deliverable.
 */
func getAppVersion(cfg Config, deliverablePath string) string {
	if strings.Contains(cfg.MavenGav.Version, "SNAPSHOT")  {
		replacer := strings.NewReplacer(cfg.MavenGav.ArtifactId, "", "-Leveransepakke.zip", "")
		version := "SNAPSHOT-" + replacer.Replace(path.Base(deliverablePath))
		return version
	}

	return cfg.MavenGav.Version
}

func getLastIndexInRepository(repository string) string {
	s := strings.Split(repository, "/")
	return s[len(s)-1]
}