package tagger

import (
	"bufio"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"sort"
	"testing"
)

const (
	CFG_PUSH_EXTRA_TAGS = "major minor patch latest"
)

const (
	APP_VERSION    = "2.4.5"
	AURORA_VERSION = "2.4.5-b1.11.0-oracle8-1.2.3"
)

const (
	TAG_MAJOR    = "2"
	TAG_MINOR    = "2.4"
	TAG_PATCH    = "2.4.5"
	TAG_COMPLETE = "2.4.5-b1.11.0-oracle8-1.2.3"
)

const (
	SNAPSHOT_GIVEN_VERSION  = "branch_test-SNAPSHOT"
	SNAPSHOT_APP_VERSION    = "branch_test-201703929219"
	SNAPSHOT_AURORA_VERSION = "SNAPSHOT-201703929219-b1.11.0-oracle8-1.2.3"
	SNAPSHOT_TAG_COMPLETE   = "SNAPSHOT-201703929219-b1.11.0-oracle8-1.2.3"
)

type RegistryMock struct {
	tagsFromRegistry []string
}

var tagger = NormalTagResolver{
	Registry:   "testregistry",
	Repository: "aurora/test",
	Provider: &RegistryMock{
		tagsFromRegistry: []string{"latest", "1.1.2", "1.1", "1", "1.2.1", "1.2", "1.3.0", "1.3", "1.1.0", "2.0.0", "2.0", "2"},
	},
}

var supertagger = NormalTagResolver{
	Registry:   "testregistry",
	Repository: "aurora/test",
	Provider: &RegistryMock{
		tagsFromRegistry: readTags(),
	},
}

func readTags() []string {
	versions := make([]string, 0, 1000)
	file, err := os.Open("versions.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		versions = append(versions, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return versions
}

func TestTagInfoRelease(t *testing.T) {
	appVersion := runtime.NewAuroraVersion(APP_VERSION, false, APP_VERSION, runtime.CompleteVersion(AURORA_VERSION))
	tags, err := tagger.ResolveTags(appVersion, config.ParseExtraTags(CFG_PUSH_EXTRA_TAGS))
	if err != nil {
		t.Fatalf("Failed to create target VersionInfo %v", err)
	}

	//TODO: Add the test for complete tag, but it should not be a part of appversion
	expectedTags := []string{"latest", TAG_MAJOR, TAG_MINOR, TAG_PATCH, TAG_COMPLETE}
	verifyTagListContent(t, tags, expectedTags)
}

func TestTagInfoSnapshot(t *testing.T) {
	appVersion := runtime.NewAuroraVersion(SNAPSHOT_APP_VERSION, true, SNAPSHOT_GIVEN_VERSION, runtime.CompleteVersion(SNAPSHOT_AURORA_VERSION))
	tags, err := tagger.ResolveTags(appVersion, config.ParseExtraTags(CFG_PUSH_EXTRA_TAGS))
	if err != nil {
		t.Fatalf("Failed to create target VersionInfo %v", err)
	}

	verifyTagListContent(t, tags, []string{SNAPSHOT_GIVEN_VERSION, SNAPSHOT_TAG_COMPLETE})
}

func TestFilterTags(t *testing.T) {
	r := repositoryTester{
		t:           t,
		tagResolver: &tagger,
	}
	r.testTagFiltering(
		runtime.NewAuroraVersion("1.1.1", false, "1.1.1", runtime.CompleteVersion("COMPLETE")),
		[]string{"1.1.1", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.2.0", false, "1.2.0", runtime.CompleteVersion("COMPLETE")),
		[]string{"1.2.0", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.2.2", false, "1.2.2", runtime.CompleteVersion("COMPLETE")),
		[]string{"1.2.2", "1.2", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.3.1", false, "1.3.1", runtime.CompleteVersion("COMPLETE")),
		[]string{"1.3.1", "1.3", "1", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("2.0.1", false, "2.0.1", runtime.CompleteVersion("COMPLETE")),
		[]string{"2.0.1", "2.0", "2", "latest", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.1.1-SNAPSHOT", true, "1.1.1-SNAPSHOT", runtime.CompleteVersion("COMPLETE")),
		[]string{"1.1.1-SNAPSHOT", "COMPLETE"})

}

func TestFilterTagsWithWeirdTagsInRepo(t *testing.T) {
	r := repositoryTester{
		t:           t,
		tagResolver: &supertagger,
	}
	r.testTagFiltering(
		runtime.NewAuroraVersion("1.106.1", false, "1.106.1", runtime.CompleteVersion("COMPLETE")),
		[]string{"1.106.1", "1.106", "1", "latest", "COMPLETE"})
}

type repositoryTester struct {
	t           *testing.T
	tagResolver TagResolver
}

func (m repositoryTester) testTagFiltering(auroraVersion *runtime.AuroraVersion, excpectedFilteringResult []string) {
	tagsToPush, err := m.tagResolver.ResolveTags(auroraVersion, config.ParseExtraTags("latest major minor patch"))
	assert.NoError(m.t, err)
	verifyTagListContent(m.t, tagsToPush, excpectedFilteringResult)
}

func (registry *RegistryMock) GetTags(repository string) (*docker.TagsAPIResponse, error) {
	return &docker.TagsAPIResponse{Name: "jalla", Tags: registry.tagsFromRegistry}, nil
}

func (registry *RegistryMock) GetImageInfo(repository string, tag string) (*runtime.ImageInfo, error) {
	return &runtime.ImageInfo{}, nil
}

func verifyTagListContent(t *testing.T, actualList []string, expectedList []string) {
	expectedListExpanded := make([]string, len(expectedList))
	for i := range expectedList {
		expectedListExpanded[i] = tagger.Registry + "/" + tagger.Repository + ":" + expectedList[i]
	}
	sort.StringSlice(expectedListExpanded).Sort()
	sort.StringSlice(actualList).Sort()
	assert.Equal(t, actualList, expectedListExpanded)
}
