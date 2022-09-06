package tagger

import (
	"context"
	"fmt"
	extVersion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"github.com/skatteetaten/architect/v2/pkg/docker"
	"github.com/skatteetaten/architect/v2/pkg/util"
	"regexp"
	"sort"
	"strings"
)

// TagResolver interface
type TagResolver interface {
	ResolveTags(appVersion *runtime.AuroraVersion, pushExtratags config.PushExtraTags) ([]string, error)
	ResolveShortTag(appVersion *runtime.AuroraVersion, pushExtratags config.PushExtraTags) ([]string, error)
}

// SingleTagResolver resolve single tag´
type SingleTagResolver struct {
	Registry   string
	Repository string
	Tag        string
}

// ResolveTags create single tag of format registry/repository:tag
func (m *SingleTagResolver) ResolveTags(_ *runtime.AuroraVersion, _ config.PushExtraTags) ([]string, error) {
	return docker.CreateImageNameFromSpecAndTags([]string{m.Tag}, m.Registry, m.Repository), nil
}

// ResolveShortTag resolve short tag e.g latest
func (m *SingleTagResolver) ResolveShortTag(_ *runtime.AuroraVersion, _ config.PushExtraTags) ([]string, error) {
	return []string{m.Tag}, nil
}

// NormalTagResolver image tag resolver
type NormalTagResolver struct {
	Registry       string
	Repository     string
	RegistryClient docker.Registry
}

// ResolveTags create tags from runtime.AuroraVersion
func (m *NormalTagResolver) ResolveTags(appVersion *runtime.AuroraVersion, pushExtratags config.PushExtraTags) ([]string, error) {
	tags, err := findCandidateTags(appVersion, m.Repository, pushExtratags, m.RegistryClient)
	if err != nil {
		return nil, err
	}
	tags = append(tags)
	return docker.CreateImageNameFromSpecAndTags(tags, m.Registry, m.Repository), nil
}

// ResolveShortTag create short tags from runtime.AuroraVersion
func (m *NormalTagResolver) ResolveShortTag(appVersion *runtime.AuroraVersion, pushExtratags config.PushExtraTags) ([]string, error) {
	tags, err := findCandidateTags(appVersion, m.Repository, pushExtratags, m.RegistryClient)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

func findCandidateTags(appVersion *runtime.AuroraVersion, outputRepository string, pushExtraTags config.PushExtraTags, provider docker.Registry) ([]string, error) {
	logrus.Debugf("Version is:%s, meta is:%s", appVersion.GetCompleteVersion(), util.GetVersionMetadata(string(appVersion.GetAppVersion())))

	if appVersion.IsSemanticReleaseVersion() {
		tagsInRepo, err := provider.GetTags(context.Background(), outputRepository)
		if err != nil {
			return nil, errors.Wrapf(err, "Error in ResolveShortTag, repository=%s", outputRepository)
		}
		logrus.Debug("Tags in repository ", tagsInRepo.Tags)

		logrus.Debugf("%s is semantic version. Filter tags", string(appVersion.GetAppVersion()))
		candidateTags, err := getSemanticVersionTags(appVersion, pushExtraTags)
		if err != nil {
			return nil, errors.Wrapf(err, "Error in FilterVersionTags, app_version=%v, repositoryTags=%v",
				appVersion, tagsInRepo.Tags)
		}
		filteredTags, err := filterTagsFromRepository(appVersion, candidateTags, tagsInRepo.Tags)
		if err != nil {
			return nil, errors.Wrapf(err, "Error in FilterVersionTags, app_version=%v, "+
				"candidateTags=%v, repositoryTags=%v", appVersion, candidateTags, tagsInRepo.Tags)
		}
		return filteredTags, nil
	}

	logrus.Debug("Is not semantic version. Append only complete version and given version")
	var versions []string
	if appVersion.Snapshot {
		if appVersion.GetUniqueSnapshotVersion() != "" {
			versions = append(versions, appVersion.GetUniqueSnapshotVersion())
		}
		versions = append(versions, appVersion.GetGivenVersion())
		versions = append(versions, appVersion.GetCompleteSnapshotVersion())
	} else {
		versions = append(versions, appVersion.GetCompleteVersion())
	}
	return versions, nil
}

func filterTagsFromRepository(version *runtime.AuroraVersion, candidateTags []string,
	repositoryTags []string) ([]string, error) {

	var excludeMinor, excludeMajor, excludeLatest = true, true, true

	minorTagName, err := getMinor(string(version.GetAppVersion()), true)
	if err != nil {
		return nil, err
	}
	excludeMinor, err = tagCompare("> "+string(version.GetAppVersion())+", < "+minorTagName, repositoryTags, version)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("Minor tag name: %s. Exclude: %t", minorTagName, excludeMinor)

	majorTagName, err := getMajor(string(version.GetAppVersion()), true)
	if err != nil {
		return nil, err
	}

	excludeMajor, err = tagCompare("> "+string(version.GetAppVersion())+", < "+majorTagName, repositoryTags, version)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("Major tag name: %s. Exclude: %t", majorTagName, excludeMajor)

	// If meta in tag we exlude latest.
	if !util.IsSemanticVersionWithMeta(string(version.GetAppVersion())) {
		excludeLatest, err = tagCompare("> "+string(version.GetAppVersion()), repositoryTags, version)
		if err != nil {
			return nil, err
		}
	}
	logrus.Debugf("Exclude latest: %t", excludeLatest)

	versions := make([]string, 0, 10)
	sort.StringSlice(candidateTags).Sort()
	for _, tag := range candidateTags {
		logrus.Debugf("Looping and checking candidate tag %s", tag)
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

func tagCompare(versionConstraint string, tags []string, version *runtime.AuroraVersion) (bool, error) {
	c, err := extVersion.NewConstraint(versionConstraint)

	if err != nil {
		return false, errors.Wrapf(err, "Could not create version constraint %s", versionConstraint)
	}
	for _, tag := range tags {
		if !util.IsSemanticVersion(tag) {
			continue
		}
		// At this point in time we only check meta vs meta. This does not comply to normal semver versioning
		if util.GetVersionMetadata(tag) != util.GetVersionMetadata(string(version.GetAppVersion())) {
			continue
		}
		v, err := extVersion.NewVersion(tag)

		if err != nil {
			return false, errors.Wrapf(err, "Error parsing version %s", tag)
		}

		if c.Check(v) {
			return true, nil
		}
	}

	return false, nil
}

func getSemanticVersionTags(version *runtime.AuroraVersion, extraTags config.PushExtraTags) ([]string, error) {
	versions := make([]string, 0, 10)

	if extraTags.Latest {
		versions = append(versions, "latest")
	}

	if extraTags.Major {
		majorVersion, err := getMajor(string(version.GetAppVersion()), false)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get major version")
		}
		versions = append(versions, majorVersion)
	}
	if extraTags.Minor {
		minorVersion, err := getMinor(string(version.GetAppVersion()), false)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get minor version")
		}
		versions = append(versions, minorVersion)
	}
	if extraTags.Patch {
		versions = append(versions, string(version.GetAppVersion()))
	}

	versions = append(versions, version.GetCompleteVersion())

	return versions, nil
}

func getMajor(version string, increment bool) (string, error) {
	buildVersion, err := extVersion.NewVersion(version)

	if err != nil {
		return "", errors.Wrap(err, "Error in parsing major version: "+version)
	}
	versionMajor := buildVersion.Segments()[0]
	if increment {
		versionMajor++
	}
	if len(buildVersion.Metadata()) > 0 {
		return fmt.Sprintf("%d+%s", versionMajor, buildVersion.Metadata()), nil
	}
	return fmt.Sprintf("%d", versionMajor), nil
}

func getMinor(version string, increment bool) (string, error) {
	buildVersion, err := extVersion.NewVersion(version)

	if err != nil {
		return "", errors.Wrap(err, "Error in parsing minor version: "+version)
	}

	versionMinor := buildVersion.Segments()[1]
	if increment {
		versionMinor++
	}
	if len(buildVersion.Metadata()) > 0 {
		return fmt.Sprintf("%d.%d+%s", buildVersion.Segments()[0], versionMinor, buildVersion.Metadata()), nil
	}
	return fmt.Sprintf("%d.%d", buildVersion.Segments()[0], versionMinor), nil
}

func isMinor(version string) bool {
	var validStr = regexp.MustCompile(`^[0-9]+.[0-9]+$`)
	versionOnly := util.GetVersionWithoutMetadata(version)
	if validStr.MatchString(versionOnly) {
		return true
	}
	return false
}

func isMajor(version string) bool {
	var validStr = regexp.MustCompile(`^[0-9]+$`)
	versionOnly := util.GetVersionWithoutMetadata(version)
	if validStr.MatchString(versionOnly) {
		return true
	}
	return false
}
