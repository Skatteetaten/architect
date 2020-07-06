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
	"strings"
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

var tagger = NormalTagResolver{
	Registry:   "testregistry",
	Repository: "aurora/test",
	Provider: &RegistryMock{
		tagsFromRegistry: []string{"latest", "1.1.2", "1.1", "1", "1.2.1", "1.2", "1.3.0",
			"1.3", "1.1.0", "2.0.0", "2.0", "2", "3+metadata", "3.2+metadata", "3.2.1+metadata"},
	},
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

var taggerWithMeta = NormalTagResolver{
	Registry:   "testregistry",
	Repository: "aurora/test",
	Provider: &RegistryMock{
		tagsFromRegistry: []string{"latest", "1.1.2", "1.1", "1", "1.2.1", "1.2", "1.3.0",
			"1.3", "1.1.0", "2.0.0", "2.0", "2", "3+metadata", "3.2+metadata", "3.2.1+metadata",
			"2+meta2", "2.0+meta2", "2.0.0+meta2"},
	},
}

/*
type NormalTagResolver struct {
	Registry   string
	Repository string
	Overwrite  bool
	Provider   docker.Registry
}
*/

func TestFilterTagsWithMeta(t *testing.T) {
	r := repositoryTester{
		t:           t,
		tagResolver: &taggerWithMeta,
	}

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.3.1", false, "1.3.1", runtime.CompleteVersion("COMPLETE")),
		[]string{"1.3.1", "1.3", "1", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.3.0", false, "1.3.0", runtime.CompleteVersion("COMPLETE")),
		[]string{"1.3.0", "1.3", "1", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.1.1", false, "1.1.1", runtime.CompleteVersion("COMPLETE")),
		[]string{"1.1.1", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.2.1", false, "1.2.1", runtime.CompleteVersion("COMPLETE")),
		[]string{"1.2.1", "1.2", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.1.1+metadata", false, "1.1.1+metadata", runtime.CompleteVersion("COMPLETE")),
		[]string{"1.1.1_metadata", "1.1_metadata", "1_metadata", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.3.0+metadata", false, "1.3.0+metadata", runtime.CompleteVersion("COMPLETE")),
		[]string{"1.3.0_metadata", "1.3_metadata", "1_metadata", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("3.2.0+metadata", false, "3.2.0+metadata", runtime.CompleteVersion("COMPLETE")),
		[]string{"3.2.0_metadata", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("3.2.2+metadata", false, "3.2.2+metadata", runtime.CompleteVersion("COMPLETE")),
		[]string{"3.2.2_metadata", "3.2_metadata", "3_metadata", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("3.2.1+metadata", false, "3.2.1+metadata", runtime.CompleteVersion("COMPLETE")),
		[]string{"3.2.1_metadata", "3.2_metadata", "3_metadata", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("3.2.1+meta+data", false, "3.2.1+meta+data", runtime.CompleteVersion("COMPLETE")),
		[]string{"COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("3.2.1+meta+data", false, "3.2.1+meta+data", runtime.CompleteVersion("COMPLETE")),
		[]string{"COMPLETE"})
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

/* Use the Append Mock to simulate a sequence of tags creation */
type RegistryMockAppend struct {
	tagsFromRegistry []string
}

var tagsAppend = []string{""}

var taggerAppend = NormalTagResolver{
	Registry:   "testregistry",
	Repository: "aurora/test",
	Provider: &RegistryMockAppend{
		tagsFromRegistry: tagsAppend,
	},
}

func TestFilterTagsWithAppend(t *testing.T) {
	r := repositoryTester{
		t:           t,
		tagResolver: &taggerAppend,
	}

	tags := r.testTagFilteringAppend("1.3.1", "COMPLETE", []string{"1.3.1", "1.3", "1", "latest", "COMPLETE"}, []string{""})
	tags = r.testTagFilteringAppend("1.3.0", "COMPLETE", []string{"1.3.0", "COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("1.3.0+meta", "COMPLETE", []string{"1.3.0+meta", "1.3+meta", "1+meta", "COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("1.3.3+meta", "COMPLETE", []string{"1.3.3+meta", "1.3+meta", "1+meta", "COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("1.3.2+meta", "COMPLETE", []string{"1.3.2+meta", "COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("1.3.2+metaannet", "COMPLETE", []string{"1.3.2+metaannet", "1.3+metaannet", "1+metaannet", "COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("2.0.0+meta-og-annet", "COMPLETE", []string{"COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("1.3.2+meta.og.annet", "COMPLETE", []string{"COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("2.0.0+meta_og_annet", "COMPLETE", []string{"COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("1.3.2+meta+og+annet", "COMPLETE", []string{"COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("1.3.2+b123meta", "COMPLETE", []string{"1.3.2+b123meta", "1.3+b123meta", "1+b123meta", "COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("1.3.1+b123meta", "COMPLETE", []string{"1.3.1+b123meta", "COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("1.3.1.0+b123meta", "COMPLETE", []string{"COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("1.b3.1+b123meta", "COMPLETE", []string{"COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("v1.3.1+b123meta", "COMPLETE", []string{"COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("1.3.1+b123?meta", "COMPLETE", []string{"COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("1.3.1+b123#meta", "COMPLETE", []string{"COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("1.4.0+b123meta", "COMPLETE", []string{"1.4.0+b123meta", "1.4+b123meta", "1+b123meta", "COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("1.2.0+b123meta", "COMPLETE", []string{"1.2.0+b123meta", "1.2+b123meta", "COMPLETE"}, tags)
	tags = r.testTagFilteringAppend("meta+1.2.3", "COMPLETE", []string{"COMPLETE"}, tags)
}

func convertTagsToRepositoryTags(tags []string) []string {
	newTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		newTags = append(newTags, docker.ConvertTagToRepositoryTag(tag))
	}
	return newTags
}

func (m repositoryTester) testTagFilteringAppend(appversion string, completeversion string, excpectedFilteringResult []string, tags []string) []string {
	tagsAppend = append(tagsAppend, tags...)
	auroraVersion := runtime.NewAuroraVersion(appversion, false, appversion, runtime.CompleteVersion(completeversion))
	tagsToPush, err := m.tagResolver.ResolveTags(auroraVersion, config.ParseExtraTags("latest major minor patch"))
	assert.NoError(m.t, err)
	verifyTagListContent(m.t, tagsToPush, convertTagsToRepositoryTags(excpectedFilteringResult))
	tagsOnly := make([]string, 0, len(tagsToPush))
	for _, tag := range tagsToPush {
		tagsOnly = append(tagsOnly, strings.Split(tag, ":")[1])
	}
	return tagsOnly
}

func (registry *RegistryMock) GetTags(repository string) (*docker.TagsAPIResponse, error) {
	return &docker.TagsAPIResponse{Name: "jalla", Tags: registry.tagsFromRegistry}, nil
}

func (registry *RegistryMock) GetManifest(repository string, digest string) (*docker.ManifestV2, error) {
	return nil, nil
}

func (registry *RegistryMock) LayerExists(repository string, layerDigest string) (bool, error) {
	return false, nil
}
func (registry *RegistryMock) MountLayer(srcRepository string, dstRepository string, layerDigest string) error {
	return nil
}
func (registry *RegistryMock) PushLayer(file string, dstRepository string, layerDigest string) error {
	return nil
}
func (registry *RegistryMock) PushManifest(file string, repository string, tag string) error {
	return nil
}

func (registry *RegistryMockAppend) GetManifest(repository string, digest string) (*docker.ManifestV2, error) {
	return nil, nil
}

func (registry *RegistryMockAppend) LayerExists(repository string, layerDigest string) (bool, error) {
	return false, nil
}
func (registry *RegistryMockAppend) MountLayer(srcRepository string, dstRepository string, layerDigest string) error {
	return nil
}
func (registry *RegistryMockAppend) PushLayer(file string, dstRepository string, layerDigest string) error {
	return nil
}
func (registry *RegistryMockAppend) PushManifest(file string, repository string, tag string) error {
	return nil
}

func (registry *RegistryMock) GetContainerConfig(repository string, digest string) (*docker.ContainerConfig, error) {
	return nil, nil
}

func (registry *RegistryMockAppend) GetContainerConfig(repository string, digest string) (*docker.ContainerConfig, error) {
	return nil, nil
}

func (registry *RegistryMockAppend) GetTags(repository string) (*docker.TagsAPIResponse, error) {
	return &docker.TagsAPIResponse{Name: "jalla", Tags: docker.ConvertRepositoryTagsToTags(tagsAppend)}, nil
}

func (registry *RegistryMock) GetImageInfo(repository string, tag string) (*runtime.ImageInfo, error) {
	return &runtime.ImageInfo{}, nil
}

func (registry *RegistryMockAppend) GetImageInfo(repository string, tag string) (*runtime.ImageInfo, error) {
	return &runtime.ImageInfo{}, nil
}

func (registry *RegistryMock) GetImageConfig(repository string, digest string) (map[string]interface{}, error) {
	return nil, nil
}

func (registry *RegistryMockAppend) GetImageConfig(repository string, digest string) (map[string]interface{}, error) {
	return nil, nil
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
