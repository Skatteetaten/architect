package runtime

import (
	"fmt"
	extVersion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"regexp"
	"strings"
)

// The Docker naming scheme sucks..
// https://docs.docker.com/glossary/?term=repository

type DockerImage struct {
	Tag        string
	Repository string
	Registry   string
}

func (m *DockerImage) GetCompleteDockerTagName() string {
	if m.Registry == "" {
		return m.Repository + ":" + m.Tag
	} else {
		return m.Registry + "/" + m.Repository + ":" + m.Tag
	}
}

//We only care of the version of architect image.. Refactor to a version variable?
type ArchitectImage struct {
	Tag string
}

/*
  Create aurora version aka complete version
  <application-version>-<builder-version>-<baseimage-repository>-<baseimage-version>
  e.g. 2.0.0-b1.11.0-oracle8-1.0.2
*/
func getCompleteVersion(appversion AppVersion, buildImage *ArchitectImage, baseImage DockerImage) string {
	return fmt.Sprintf("%s-%s-%s", appversion,
		buildImage.AuroraVersionComponent(),
		baseImage.AuroraVersionComponent())
}

func (m *DockerImage) AuroraVersionComponent() string {
	//TODO: Can we assume that we have two one components?
	s := strings.Split(m.Repository, "/")
	return s[len(s)-1] + "-" + m.Tag
}

func (m *ArchitectImage) AuroraVersionComponent() string {
	//TODO: Should we include image name?
	return "b" + m.Tag
}

type AppVersion string

type GivenVersion string

type CompleteVersion string

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
type AuroraVersion struct {
	appVersion AppVersion // The actual version of the artifact. This will differ from GivenVersion iff the artifact
	// is a snapshot artifact (the SNAPSHOT-part will be replaced with the timestamp generated by Nexus)
	Snapshot        bool         // If the artifact is a snapshot artifact. NPM has different semantics, so we are explicit
	givenVersion    GivenVersion // The version set in the build config
	completeVersion CompleteVersion
}

func NewAuroraVersion(appVersion string, snapshot bool, givenVersion string, completeVersion CompleteVersion) *AuroraVersion {
	return &AuroraVersion{
		appVersion:      AppVersion(appVersion),
		Snapshot:        snapshot,
		givenVersion:    GivenVersion(givenVersion),
		completeVersion: completeVersion,
	}
}

func NewAuroraVersionFromBuilderAndBase(
	appVersion string, snapshot bool,
	givenVersion string, buildImage *ArchitectImage, baseImage DockerImage) *AuroraVersion {
	return &AuroraVersion{
		appVersion:      AppVersion(appVersion),
		Snapshot:        snapshot,
		givenVersion:    GivenVersion(givenVersion),
		completeVersion: CompleteVersion(getCompleteVersion(AppVersion(appVersion), buildImage, baseImage)),
	}
}

func (m *AuroraVersion) GetGivenVersion() string {
	return string(m.givenVersion)
}

func (m *AuroraVersion) GetCompleteVersion() string {
	return string(m.completeVersion)
}

func (m *AuroraVersion) GetAppVersion() AppVersion {
	return m.appVersion
}

func (m *AuroraVersion) getSemtanticVersion(extraTags config.PushExtraTags) ([]string, error) {
	versions := make([]string, 0, 10)

	if extraTags.Latest {
		versions = append(versions, "latest")
	}

	if extraTags.Major {
		majorVersion, err := getMajor(string(m.appVersion), false)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get major version")
		}
		versions = append(versions, majorVersion)
	}
	if extraTags.Minor {
		minorVersion, err := getMinor(string(m.appVersion), false)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get minor version")
		}
		versions = append(versions, minorVersion)
	}
	if extraTags.Patch {
		versions = append(versions, string(m.appVersion))
	}

	versions = append(versions, string(m.completeVersion))

	return versions, nil
}

func (m *AuroraVersion) GetApplicationVersionTagsToPush(repositoryTags []string, extraTags config.PushExtraTags) ([]string, error) {
	versions := make([]string, 0, 10)
	if !m.isSemanticReleaseVersion() {
		versions = append(versions, string(m.completeVersion))
		if m.Snapshot {
			versions = append(versions, string(m.givenVersion))
		}
		return versions, nil
	} else {
		newTags, err := m.getSemtanticVersion(extraTags)
		if err != nil {
			return nil, err
		}
		var excludeMinor, excludeMajor, excludeLatest bool = true, true, true

		minorTagName, err := getMinor(string(m.appVersion), true)
		if err != nil {
			return nil, err
		}

		excludeMinor, err = tagCompare("> "+string(m.appVersion)+", < "+minorTagName, repositoryTags)
		if err != nil {
			return nil, err
		}

		majorTagName, err := getMajor(string(m.appVersion), true)
		if err != nil {
			return nil, err
		}

		excludeMajor, err = tagCompare("> "+string(m.appVersion)+", < "+majorTagName, repositoryTags)
		if err != nil {
			return nil, err
		}

		excludeLatest, err = tagCompare("> "+string(m.appVersion), repositoryTags)
		if err != nil {
			return nil, err
		}

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
	}
	return versions, nil
}

func tagCompare(versionConstraint string, tags []string) (bool, error) {
	c, err := extVersion.NewConstraint(versionConstraint)

	if err != nil {
		return false, errors.Wrapf(err, "Could not create version constraint %s", versionConstraint)
	}

	for _, tag := range tags {
		v, err := extVersion.NewVersion(tag)

		if err != nil {
			// We won't fail on random tags in the reposiority
			return false, nil
		}

		if c.Check(v) {
			return true, nil
		}
	}

	return false, nil
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

//TODO: Snapshot / Semantic? Whats the difference?
func (m *AuroraVersion) isSemanticReleaseVersion() bool {
	if m.Snapshot {
		return false
	}
	var validStr = regexp.MustCompile(`^[0-9]+.[0-9]+.[0-9]+$`)
	if validStr.MatchString(string(m.appVersion)) {
		return true
	}
	return false
}
