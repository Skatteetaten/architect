package runtime

import (
	"fmt"
	"github.com/skatteetaten/architect/v2/pkg/util"
	"strings"
)

// ImageInfo struct containing image metadata
type ImageInfo struct {
	CompleteBaseImageVersion string
	Labels                   map[string]string
	Environment              map[string]string
	Digest                   string
}

// BaseImage struct containing base image metadata
type BaseImage struct {
	DockerImage
	ImageInfo *ImageInfo
}

// DockerImage tag representation
type DockerImage struct {
	Tag        string
	Repository string
	Registry   string
}

// GetCompleteDockerTagName registry/repository:tag
func (m *DockerImage) GetCompleteDockerTagName() string {
	if m.Registry == "" {
		return m.Repository + ":" + m.Tag
	}

	return m.Registry + "/" + m.Repository + ":" + m.Tag
}

// ArchitectImage tag
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

// AuroraVersionComponent return the aurora version component
func (m *DockerImage) AuroraVersionComponent() string {
	//TODO: Can we assume that we have two one components?
	s := strings.Split(m.Repository, "/")
	return s[len(s)-1] + "-" + m.Tag
}

// AuroraVersionComponent return architect version component
func (m *ArchitectImage) AuroraVersionComponent() string {
	return "b" + m.Tag
}

// AppVersion type string
type AppVersion string

// GivenVersion type string
type GivenVersion string

// CompleteVersion type string
type CompleteVersion string

// AuroraVersion EKSEMPEL:
//
// Gitt følgende URL http://uil0map-paas-app01.skead.no:9090/v2/aurora/console/tags/list
// COMPLETE	 		    = 2.0.0-b1.11.0-oracle8-1.0.2
// LATEST				= latest
// MAJOR				= 2
// MINOR				= 2.0
// PATCH				= 2.0.0
// OutputImage.Repository	aurora/console
type AuroraVersion struct {
	appVersion AppVersion // The actual version of the artifact. This will differ from GivenVersion iff the artifact
	// is a snapshot artifact (the SNAPSHOT-part will be replaced with the timestamp generated by Nexus)
	Snapshot                bool         // If the artifact is a snapshot artifact. NPM has different semantics, so we are explicit
	givenVersion            GivenVersion // The version set in the build config
	completeVersion         CompleteVersion
	completeSnapshotVersion CompleteVersion
	hash                    string
}

// NewAuroraVersion new AuroraVersion
func NewAuroraVersion(appVersion string, snapshot bool, givenVersion string, completeVersion CompleteVersion) *AuroraVersion {
	return &AuroraVersion{
		appVersion:              AppVersion(appVersion),
		Snapshot:                snapshot,
		givenVersion:            GivenVersion(givenVersion),
		completeVersion:         completeVersion,
		completeSnapshotVersion: completeVersion,
	}
}

// NewAuroraVersionFromBuilderAndBase return new version with builder and base version
func NewAuroraVersionFromBuilderAndBase(
	appVersion string, snapshot bool,
	givenVersion string, buildImage *ArchitectImage, baseImage DockerImage, hash string) *AuroraVersion {
	return &AuroraVersion{
		appVersion:              AppVersion(appVersion),
		Snapshot:                snapshot,
		givenVersion:            GivenVersion(givenVersion),
		completeVersion:         CompleteVersion(getCompleteVersion(AppVersion(appVersion), buildImage, baseImage)),
		completeSnapshotVersion: CompleteVersion(getCompleteVersion(AppVersion(givenVersion), buildImage, baseImage)),
		hash:                    hash,
	}
}

// GetGivenVersion returns the version set in the build config
func (m *AuroraVersion) GetGivenVersion() string {
	return string(m.givenVersion)
}

// GetCompleteVersion 2.0.0-b1.11.0-oracle8-1.0.2
func (m *AuroraVersion) GetCompleteVersion() string {
	return string(m.completeVersion)
}

// GetCompleteSnapshotVersion test-snapshot-b1.11.0-oracle8-1.0.2
func (m *AuroraVersion) GetCompleteSnapshotVersion() string {
	return string(m.completeSnapshotVersion)
}

// GetAppVersion The actual version of the artifact. This will differ from GivenVersion iff the artifact
func (m *AuroraVersion) GetAppVersion() AppVersion {
	return m.appVersion
}

// GetUniqueSnapshotVersion snapshot version with appended hash
func (m *AuroraVersion) GetUniqueSnapshotVersion() string {
	if m.hash == "" || len(m.hash) < 8 {
		return ""
	}
	return fmt.Sprintf("%s-%s", m.GetGivenVersion(), m.hash[len(m.hash)-8:])
}

// IsSemanticReleaseVersion check if version is a semantic version
func (m *AuroraVersion) IsSemanticReleaseVersion() bool {
	if m.Snapshot {
		return false
	}
	return util.IsFullSemanticVersion(string(m.appVersion))
}
