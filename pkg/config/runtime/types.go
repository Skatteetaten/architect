package runtime

import (
	"fmt"
	"github.com/skatteetaten/architect/pkg/util"
	"strings"
)

type ImageInfo struct {
	CompleteBaseImageVersion string
	Labels                   map[string]string
	Enviroment               map[string]string
	Digest                   string
}

type BaseImage struct {
	DockerImage
	ImageInfo *ImageInfo
}

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
	}

	return m.Registry + "/" + m.Repository + ":" + m.Tag
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
	Snapshot                bool         // If the artifact is a snapshot artifact. NPM has different semantics, so we are explicit
	givenVersion            GivenVersion // The version set in the build config
	completeVersion         CompleteVersion
	completeSnapshotVersion CompleteVersion
	hash                    string
}

func NewAuroraVersion(appVersion string, snapshot bool, givenVersion string, completeVersion CompleteVersion) *AuroraVersion {
	return &AuroraVersion{
		appVersion:              AppVersion(appVersion),
		Snapshot:                snapshot,
		givenVersion:            GivenVersion(givenVersion),
		completeVersion:         completeVersion,
		completeSnapshotVersion: completeVersion,
	}
}

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

func (m *AuroraVersion) GetGivenVersion() string {
	return string(m.givenVersion)
}

func (m *AuroraVersion) GetCompleteVersion() string {
	return string(m.completeVersion)
}

func (m *AuroraVersion) GetCompleteSnapshotVersion() string {
	return string(m.completeSnapshotVersion)
}

func (m *AuroraVersion) GetAppVersion() AppVersion {
	return m.appVersion
}

func (m *AuroraVersion) GetUniqueSnapshotVersion() string {
	if m.hash == "" || len(m.hash) < 8 {
		return ""
	}
	return fmt.Sprintf("%s-%s", m.GetGivenVersion(), m.hash[len(m.hash)-8:])
}

//TODO: Snapshot / Semantic? Whats the difference?
func (m *AuroraVersion) IsSemanticReleaseVersion() bool {
	if m.Snapshot {
		return false
	}
	return util.IsFullSemanticVersion(string(m.appVersion))
}
