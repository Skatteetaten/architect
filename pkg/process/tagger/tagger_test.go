package tagger

import (
	"bufio"
	"context"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"github.com/skatteetaten/architect/v2/pkg/docker"
	"github.com/stretchr/testify/assert"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"testing"
)

const (
	CfgPushExtraTags = "major minor patch latest"
)

const (
	AppVersion    = "2.4.5"
	AuroraVersion = "2.4.5-b1.11.0-oracle8-1.2.3"
)

const (
	TagMajor    = "2"
	TagMinor    = "2.4"
	TagPatch    = "2.4.5"
	TagComplete = "2.4.5-b1.11.0-oracle8-1.2.3"
)

const (
	SnapshotGivenVersion  = "branch_test-SNAPSHOT"
	SnapshotAppVersion    = "branch_test-201703929219"
	SnapshotAuroraVersion = "SNAPSHOT-201703929219-b1.11.0-oracle8-1.2.3"
	SnapshotTagComplete   = "SNAPSHOT-201703929219-b1.11.0-oracle8-1.2.3"
)

type RegistryMock struct {
	tagsFromRegistry []string
}

var supertagger = NormalTagResolver{
	Registry:   "testregistry",
	Repository: "aurora/test",
	RegistryClient: &RegistryMock{
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
	RegistryClient: &RegistryMock{
		tagsFromRegistry: []string{"latest", "1.1.2", "1.1", "1", "1.2.1", "1.2", "1.3.0",
			"1.3", "1.1.0", "2.0.0", "2.0", "2", "3+metadata", "3.2+metadata", "3.2.1+metadata"},
	},
}

func TestTagInfoRelease(t *testing.T) {
	appVersion := runtime.NewAuroraVersion(AppVersion, false, AppVersion, AuroraVersion)
	tags, err := tagger.ResolveTags(appVersion, config.ParseExtraTags(CfgPushExtraTags))
	if err != nil {
		t.Fatalf("Failed to create target VersionInfo %v", err)
	}

	//TODO: Add the test for complete tag, but it should not be a part of appversion
	expectedTags := []string{"latest", TagMajor, TagMinor, TagPatch, TagComplete}
	verifyTagListContent(t, tags, expectedTags)
}

func TestTagInfoSnapshot(t *testing.T) {
	appVersion := runtime.NewAuroraVersion(SnapshotAppVersion, true, SnapshotGivenVersion, SnapshotAuroraVersion)
	tags, err := tagger.ResolveTags(appVersion, config.ParseExtraTags(CfgPushExtraTags))
	if err != nil {
		t.Fatalf("Failed to create target VersionInfo %v", err)
	}

	verifyTagListContent(t, tags, []string{SnapshotGivenVersion, SnapshotTagComplete})
}

func TestFilterTags(t *testing.T) {
	r := repositoryTester{
		t:           t,
		tagResolver: &tagger,
	}

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.1.1", false, "1.1.1", "COMPLETE"),
		[]string{"1.1.1", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.2.0", false, "1.2.0", "COMPLETE"),
		[]string{"1.2.0", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.2.2", false, "1.2.2", "COMPLETE"),
		[]string{"1.2.2", "1.2", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.3.1", false, "1.3.1", "COMPLETE"),
		[]string{"1.3.1", "1.3", "1", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("2.0.1", false, "2.0.1", "COMPLETE"),
		[]string{"2.0.1", "2.0", "2", "latest", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.1.1-SNAPSHOT", true, "1.1.1-SNAPSHOT", "COMPLETE"),
		[]string{"1.1.1-SNAPSHOT", "COMPLETE"})
}

var taggerWithMeta = NormalTagResolver{
	Registry:   "testregistry",
	Repository: "aurora/test",
	RegistryClient: &RegistryMock{
		tagsFromRegistry: []string{"latest", "1.1.2", "1.1", "1", "1.2.1", "1.2", "1.3.0",
			"1.3", "1.1.0", "2.0.0", "2.0", "2", "3+metadata", "3.2+metadata", "3.2.1+metadata",
			"2+meta2", "2.0+meta2", "2.0.0+meta2"},
	},
}

func TestFilterTagsWithMeta(t *testing.T) {
	r := repositoryTester{
		t:           t,
		tagResolver: &taggerWithMeta,
	}

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.3.1", false, "1.3.1", "COMPLETE"),
		[]string{"1.3.1", "1.3", "1", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.3.0", false, "1.3.0", "COMPLETE"),
		[]string{"1.3.0", "1.3", "1", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.1.1", false, "1.1.1", "COMPLETE"),
		[]string{"1.1.1", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.2.1", false, "1.2.1", "COMPLETE"),
		[]string{"1.2.1", "1.2", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.1.1+metadata", false, "1.1.1+metadata", "COMPLETE"),
		[]string{"1.1.1_metadata", "1.1_metadata", "1_metadata", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("1.3.0+metadata", false, "1.3.0+metadata", "COMPLETE"),
		[]string{"1.3.0_metadata", "1.3_metadata", "1_metadata", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("3.2.0+metadata", false, "3.2.0+metadata", "COMPLETE"),
		[]string{"3.2.0_metadata", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("3.2.2+metadata", false, "3.2.2+metadata", "COMPLETE"),
		[]string{"3.2.2_metadata", "3.2_metadata", "3_metadata", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("3.2.1+metadata", false, "3.2.1+metadata", "COMPLETE"),
		[]string{"3.2.1_metadata", "3.2_metadata", "3_metadata", "COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("3.2.1+meta+data", false, "3.2.1+meta+data", "COMPLETE"),
		[]string{"COMPLETE"})

	r.testTagFiltering(
		runtime.NewAuroraVersion("3.2.1+meta+data", false, "3.2.1+meta+data", "COMPLETE"),
		[]string{"COMPLETE"})
}

func TestFilterTagsWithWeirdTagsInRepo(t *testing.T) {
	r := repositoryTester{
		t:           t,
		tagResolver: &supertagger,
	}
	r.testTagFiltering(
		runtime.NewAuroraVersion("1.106.1", false, "1.106.1", "COMPLETE"),
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
	RegistryClient: &RegistryMockAppend{
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

func (registry *RegistryMock) GetTags(ctx context.Context, repository string) (*docker.TagsAPIResponse, error) {
	return &docker.TagsAPIResponse{Name: "jalla", Tags: registry.tagsFromRegistry}, nil
}

func (registry *RegistryMock) GetManifest(ctx context.Context, repository string, digest string) (*docker.ManifestV2, error) {
	return nil, nil
}

func (registry *RegistryMock) LayerExists(ctx context.Context, repository string, layerDigest string) (bool, error) {
	return false, nil
}
func (registry *RegistryMock) MountLayer(ctx context.Context, srcRepository string, dstRepository string, layerDigest string) error {
	return nil
}
func (registry *RegistryMock) PushLayer(ctx context.Context, layer io.Reader, dstRepository string, layerDigest string) error {
	return nil
}

func (registry *RegistryMock) PullLayer(ctx context.Context, repository string, layerDigest string) (string, error) {
	return "", nil
}

func (registry *RegistryMock) PushManifest(ctx context.Context, manifest []byte, repository string, tag string) error {
	return nil
}

func (registry *RegistryMockAppend) GetManifest(ctx context.Context, repository string, digest string) (*docker.ManifestV2, error) {
	return nil, nil
}

func (registry *RegistryMockAppend) LayerExists(ctx context.Context, repository string, layerDigest string) (bool, error) {
	return false, nil
}
func (registry *RegistryMockAppend) MountLayer(ctx context.Context, srcRepository string, dstRepository string, layerDigest string) error {
	return nil
}
func (registry *RegistryMockAppend) PushLayer(ctx context.Context, layer io.Reader, dstRepository string, layerDigest string) error {
	return nil
}

func (registry *RegistryMockAppend) PullLayer(ctx context.Context, repository string, layerDigest string) (string, error) {
	return "", nil
}

func (registry *RegistryMockAppend) PushManifest(ctx context.Context, manifest []byte, repository string, tag string) error {
	return nil
}

func (registry *RegistryMock) GetContainerConfig(ctx context.Context, repository string, digest string) (*docker.ContainerConfig, error) {
	return nil, nil
}

func (registry *RegistryMockAppend) GetContainerConfig(ctx context.Context, repository string, digest string) (*docker.ContainerConfig, error) {
	return nil, nil
}

func (registry *RegistryMockAppend) GetTags(ctx context.Context, repository string) (*docker.TagsAPIResponse, error) {
	return &docker.TagsAPIResponse{Name: "jalla", Tags: docker.ConvertRepositoryTagsToTags(tagsAppend)}, nil
}

func (registry *RegistryMock) GetImageInfo(ctx context.Context, repository string, tag string) (*runtime.ImageInfo, error) {
	return &runtime.ImageInfo{}, nil
}

func (registry *RegistryMockAppend) GetImageInfo(ctx context.Context, repository string, tag string) (*runtime.ImageInfo, error) {
	return &runtime.ImageInfo{}, nil
}

func (registry *RegistryMock) GetImageConfig(ctx context.Context, repository string, digest string) (map[string]interface{}, error) {
	return nil, nil
}

func (registry *RegistryMockAppend) GetImageConfig(ctx context.Context, repository string, digest string) (map[string]interface{}, error) {
	return nil, nil
}

func verifyTagListContent(t *testing.T, actualList []string, expectedList []string) {
	expectedListExpanded := make([]string, len(expectedList))
	for i := range expectedList {
		expectedListExpanded[i] = tagger.Registry + "/" + tagger.Repository + ":" + expectedList[i]
	}
	sort.StringSlice(expectedListExpanded).Sort()
	sort.StringSlice(actualList).Sort()
	assert.Equal(t, expectedListExpanded, actualList)
}
