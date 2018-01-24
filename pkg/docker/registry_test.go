package docker

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const repository = "aurora/flange"
const tag = "8"

func TestGetManifestEnvSchemaV1(t *testing.T) {
	expectedVersion := "1.7.0"

	server, err := startMockRegistryServer("testdata/manifest.json")

	defer server.Close()

	assert.NoError(t, err)

	target := NewRegistryClient(server.URL)

	manifestEnvMap, err := target.GetManifestEnvMap(repository, tag)
	assert.NoError(t, err)
	actualVersion := manifestEnvMap["BASE_IMAGE_VERSION"]
	assert.Equal(t, expectedVersion, actualVersion)
}

func TestGetCompleteBaseImageVersionSchemaV1(t *testing.T) {
	expectedVersion := "1.7.0"

	server, err := startMockRegistryServer("testdata/manifest.json")

	defer server.Close()

	assert.NoError(t, err)

	target := NewRegistryClient(server.URL)

	actualVersion, err := target.GetCompleteBaseImageVersion(repository, tag)
	assert.NoError(t, err)
	assert.Equal(t, expectedVersion, actualVersion)
}

func TestGetManifestEnvMapSchemaV1(t *testing.T) {
	expectedLength := 14

	server, err := startMockRegistryServer("testdata/manifest.json")

	defer server.Close()

	assert.NoError(t, err)

	target := NewRegistryClient(server.URL)

	envMap, err := target.GetManifestEnvMap(repository, tag)

	assert.NoError(t, err)

	actualLength := len(envMap)
	assert.Equal(t, expectedLength, actualLength)
}

func TestGetManifestEnvSchemaV2(t *testing.T) {
	expectedVersion := "8.152.18"

	server, err := startMockRegistryManifestServer("testdata/aurora_flange_manifest_v2.json", "testdata/aurora_flange_container_image_v1.json")

	defer server.Close()

	assert.NoError(t, err)

	target := NewRegistryClient(server.URL)

	manifestEnvMap, err := target.GetManifestEnvMap(repository, tag)
	assert.NoError(t, err)
	actualVersion := manifestEnvMap["BASE_IMAGE_VERSION"]
	assert.Equal(t, expectedVersion, actualVersion)
}

func TestGetCompleteBaseImageVersionSchemaV2(t *testing.T) {
	expectedVersion := "8.152.18"

	server, err := startMockRegistryManifestServer("testdata/aurora_flange_manifest_v2.json", "testdata/aurora_flange_container_image_v1.json")

	defer server.Close()

	assert.NoError(t, err)

	target := NewRegistryClient(server.URL)

	actualVersion, err := target.GetCompleteBaseImageVersion(repository, tag)
	assert.NoError(t, err)
	assert.Equal(t, expectedVersion, actualVersion)
}

func TestGetManifestEnvMapSchemaV2(t *testing.T) {
	expectedLength := 13

	server, err := startMockRegistryManifestServer("testdata/aurora_flange_manifest_v2.json", "testdata/aurora_flange_container_image_v1.json")

	defer server.Close()

	assert.NoError(t, err)

	target := NewRegistryClient(server.URL)

	envMap, err := target.GetManifestEnvMap(repository, tag)

	assert.NoError(t, err)

	actualLength := len(envMap)
	assert.Equal(t, expectedLength, actualLength)
}

func TestGetTags(t *testing.T) {
	server, err := startMockRegistryServer("testdata/tags.list.json")

	defer server.Close()

	assert.NoError(t, err)

	expectedTags := []string{"latest", "develop-SNAPSHOT",
		"develop-SNAPSHOT-9be2b9ca43a024415947a6c262e183406dbb090b",
		"2.0.0", "1.3.0", "1.2.1", "1.1.2", "1.1", "1.2", "1.3", "2.0", "2", "1"}

	target := NewRegistryClient(server.URL)

	tags, err := target.GetTags("aurora/oracle8")

	assert.NoError(t, err)

	verifyTagListContent(tags.Tags, expectedTags, t)
}

func verifyTagListContent(actualList []string, expectedList []string, t *testing.T) {
	if len(actualList) != len(expectedList) {
		t.Errorf("Expected %d tags, actual is %d", len(expectedList), len(actualList))
	}

	for _, e := range expectedList {
		verifyContainsTag(actualList, e, t)
	}
}

func verifyContainsTag(actual []string, expected string, t *testing.T) {
	if !contains(actual, expected) {
		t.Errorf("Expected tag %s does not exist", expected)
	}
}

func contains(target []string, value string) bool {
	for _, t := range target {
		if t == value {
			return true
		}
	}

	return false
}

func startMockRegistryServer(filename string) (*httptest.Server, error) {
	buf, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(buf)))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(buf)
	}))
	ts.StartTLS()
	return ts, nil
}

func startMockRegistryManifestServer(fileManifest string, fileImageMeta string) (*httptest.Server, error) {
	bufManifest, err := ioutil.ReadFile(fileManifest)

	if err != nil {
		return nil, err
	}

	bufImageMeta, err := ioutil.ReadFile(fileImageMeta)

	if err != nil {
		return nil, err
	}

	bufManifestError, err := ioutil.ReadFile("testdata/aurora_flange_manifest_error.json")

	if err != nil {
		return nil, err
	}

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf []byte
		if strings.Contains(r.Header.Get("Accept"), httpHeaderManifestSchemaV2) {
			buf = bufManifest
		} else if strings.Contains(r.Header.Get("Accept"), httpHeaderContainerImageV1) {
			buf = bufImageMeta
		} else {
			buf = bufManifestError
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(buf)))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(buf)
	}))
	ts.StartTLS()
	return ts, nil
}
