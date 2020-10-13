package tagger

import (
	"context"
	"fmt"
	extVersion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/util"
	"regexp"
	"sort"
	"strings"
)

type TagResolver interface {
	ResolveTags(appVersion *runtime.AuroraVersion, pushExtratags config.PushExtraTags) ([]string, error)
	ResolveShortTag(appVersion *runtime.AuroraVersion, pushExtratags config.PushExtraTags) ([]string, error)
}

type SingleTagTagResolver struct {
	Registry   string
	Repository string
	Tag        string
}

func (m *SingleTagTagResolver) ResolveTags(appVersion *runtime.AuroraVersion, pushExtratags config.PushExtraTags) ([]string, error) {
	return docker.CreateImageNameFromSpecAndTags([]string{m.Tag}, m.Registry, m.Repository), nil
}

func (m *SingleTagTagResolver) ResolveShortTag(appVersion *runtime.AuroraVersion, pushExtratags config.PushExtraTags) ([]string, error) {
	return []string{m.Tag}, nil
}

type NormalTagResolver struct {
	Registry       string
	Repository     string
	Overwrite      bool
	RegistryClient docker.Registry
}

func (m *NormalTagResolver) ResolveTags(appVersion *runtime.AuroraVersion, pushExtratags config.PushExtraTags) ([]string, error) {
	tags, err := findCandidateTags(appVersion, m.Overwrite, m.Repository, pushExtratags, m.RegistryClient)
	if err != nil {
		return nil, err
	}
	tags = append(tags)
	return docker.CreateImageNameFromSpecAndTags(tags, m.Registry, m.Repository), nil
}

func (m *NormalTagResolver) ResolveShortTag(appVersion *runtime.AuroraVersion, pushExtratags config.PushExtraTags) ([]string, error) {
	tags, err := findCandidateTags(appVersion, m.Overwrite, m.Repository, pushExtratags, m.RegistryClient)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

func findCandidateTags(appVersion *runtime.AuroraVersion, tagOverwrite bool, outputRepository string,
	pushExtraTags config.PushExtraTags, provider docker.Registry) ([]string, error) {
	var repositoryTags []string
	logrus.Debugf("Version is:%s, meta is:%s", appVersion.GetCompleteVersion(), util.GetVersionMetadata(string(appVersion.GetAppVersion())))
	if !tagOverwrite {

		tagsInRepo, err := provider.GetTags(context.Background(), outputRepository)
		if err != nil {
			return nil, errors.Wrapf(err, "Error in ResolveShortTag, repository=%s", outputRepository)
		}
		logrus.Debug("Tags in repository ", tagsInRepo.Tags)
		repositoryTags = tagsInRepo.Tags
	}
	if appVersion.IsSemanticReleaseVersion() {
		logrus.Debugf("%s is semantic version. Filter tags", string(appVersion.GetAppVersion()))
		candidateTags, err := getSemanticVersionTags(appVersion, pushExtraTags)
		if err != nil {
			return nil, errors.Wrapf(err, "Error in FilterVersionTags, app_version=%v, repositoryTags=%v",
				appVersion, repositoryTags)
		}
		filteredTags, err := filterTagsFromRepository(appVersion, candidateTags, repositoryTags)
		if err != nil {
			return nil, errors.Wrapf(err, "Error in FilterVersionTags, app_version=%v, "+
				"candidateTags=%v, repositoryTags=%v", appVersion, candidateTags, repositoryTags)
		}
		return filteredTags, nil
	} else {
		versions := make([]string, 0, 10)
		logrus.Debug("Is not semantic version. Append only complete version and given version")
		versions = append(versions, string(appVersion.GetCompleteVersion()))
		if appVersion.Snapshot {
			versions = append(versions, string(appVersion.GetGivenVersion()))
		}
		return versions, nil
	}
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

	versions = append(versions, string(version.GetCompleteVersion()))

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
	build_version, err := extVersion.NewVersion(version)

	if err != nil {
		return "", errors.Wrap(err, "Error in parsing minor version: "+version)
	}

	versionMinor := build_version.Segments()[1]
	if increment {
		versionMinor++
	}
	if len(build_version.Metadata()) > 0 {
		return fmt.Sprintf("%d.%d+%s", build_version.Segments()[0], versionMinor, build_version.Metadata()), nil
	}
	return fmt.Sprintf("%d.%d", build_version.Segments()[0], versionMinor), nil
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
