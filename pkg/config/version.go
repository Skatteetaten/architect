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
	VersionTags []string
}

type BuildInfo struct {
	Env         map[string]string
	OutputImage OutputImageInfo
	BaseImage   BaseImageInfo
}

type OutputImageInfo struct {
	Repository string
	TagInfo
}

type BaseImageInfo struct {
	Repository string
	Version    string
}

func NewTagInfo(appVersion string, auroraVersion string, extraTags string) (*TagInfo, error) {
	versionTags, err := getVersionTags(appVersion, auroraVersion, extraTags)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize builder")
	}

	return &TagInfo{VersionTags: versionTags}, nil
}

func NewBuildInfo(cfg Config, deliverable Deliverable, imageInfoProvider docker.ImageInfoProvider) (*BuildInfo, error) {

	baseImageVersion, err := getBaseImageVersion(imageInfoProvider, cfg)

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

func FilterVersionTags(appVersion string, newTags []string, repositoryTags []string) ([]string, error) {
	if !isSemantic(appVersion) {
		return newTags, nil
	}

	var excludeMinor, excludeMajor, excludeLatest bool = true, true, true

	minorTagName, err := getMinor(appVersion, true)

	if err != nil {
		return nil, err
	}

	excludeMinor, err = tagCompare("> "+appVersion+", < "+minorTagName, repositoryTags)

	if err != nil {
		return nil, err
	}

	majorTagName, err := getMajor(appVersion, true)

	if err != nil {
		return nil, err
	}

	excludeMajor, err = tagCompare("> "+appVersion+", < "+majorTagName, repositoryTags)

	if err != nil {
		return nil, err
	}

	excludeLatest, err = tagCompare("> "+appVersion, repositoryTags)

	if err != nil {
		return nil, err
	}

	versions := make([]string, 0, 10)

	for _, tag := range newTags {
		if strings.EqualFold(strings.TrimSpace(tag), "latest") {
			if !excludeLatest {
				versions = append(versions, tag)
			}
		} else if isMinor(tag) {
			if !excludeMinor {
				versions = append(versions, tag)
			}
		} else if isMajor(tag) {
			if !excludeMajor {
				versions = append(versions, tag)
			}
		} else {
			versions = append(versions, tag)
		}
	}
	return versions, nil
}

func tagCompare(versionConstraint string, tags []string) (bool, error) {
	c, err := extVersion.NewConstraint(versionConstraint)

	if err != nil {
		return false, errors.Wrapf(err, "Could not create version constraint %s", versionConstraint)
	}

	for _, tag := range tags {
		if isSemantic(tag) {
			v, err := extVersion.NewVersion(tag)

			if err != nil {
				return false, errors.Wrapf(err, "Could not create tag constraint %s", tag)
			}

			if c.Check(v) {
				return true, nil
			}
		}
	}

	return false, nil
}

func createOutputImageInfo(appVersion string, auroraVersion string, cfg Config) (*OutputImageInfo, error) {
	var tags []string
	var err error

	if !isTemporary(cfg) {
		tags, err = getVersionTags(appVersion, auroraVersion, cfg.DockerSpec.PushExtraTags)

		//TODO: This should really be inside getVersionTags. Refactor
		if isSnapshot(cfg) {
			tags = append(tags, cfg.MavenGav.Version)
		}

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

	env[docker.ENV_APP_VERSION] = appVersion
	env[docker.ENV_AURORA_VERSION] = auroraVersion
	env[docker.ENV_PUSH_EXTRA_TAGS] = cfg.DockerSpec.PushExtraTags
	env[docker.TZ] = "Europe/Oslo"

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

	versions = append(versions, auroraVersion)

	if isSemantic(appVersion) {

		if strings.Contains(extraTags, "latest") {
			versions = append(versions, "latest")
		}

		if strings.Contains(extraTags, "major") {
			majorVersion, err := getMajor(appVersion, false)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get major version")
			}

			versions = append(versions, majorVersion)
		}

		if strings.Contains(extraTags, "minor") {
			minorVersion, err := getMinor(appVersion, false)
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

func getMajor(version string, bumpVersion bool) (string, error) {
	build_version, err := extVersion.NewVersion(version)

	if err != nil {
		return "", errors.Wrap(err, "Error in parsing major version: "+version)
	}

	versionMajor := build_version.Segments()[0]
	if bumpVersion {
		versionMajor += 1
	}

	return fmt.Sprintf("%d", versionMajor), nil
}

func isMajor(version string) bool {
	var validStr = regexp.MustCompile(`^[0-9]+$`)
	if validStr.MatchString(version) {
		return true
	}
	return false
}

func getMinor(version string, bumpVersion bool) (string, error) {
	build_version, err := extVersion.NewVersion(version)

	if err != nil {
		return "", errors.Wrap(err, "Error in parsing minor version: "+version)
	}

	versionMinor := build_version.Segments()[1]
	if bumpVersion {
		versionMinor += 1
	}

	return fmt.Sprintf("%d.%d", build_version.Segments()[0], versionMinor), nil
}

func isMinor(version string) bool {
	var validStr = regexp.MustCompile(`^[0-9]+.[0-9]+$`)
	if validStr.MatchString(version) {
		return true
	}
	return false
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

func getBaseImageVersion(provider docker.ImageInfoProvider, config Config) (string, error) {
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
	if strings.Contains(cfg.MavenGav.Version, "SNAPSHOT") {
		replacer := strings.NewReplacer(cfg.MavenGav.ArtifactId, "", "-Leveransepakke.zip", "")
		version := "SNAPSHOT" + replacer.Replace(path.Base(deliverablePath))
		return version
	}

	return cfg.MavenGav.Version
}

func getLastIndexInRepository(repository string) string {
	s := strings.Split(repository, "/")
	return s[len(s)-1]
}
