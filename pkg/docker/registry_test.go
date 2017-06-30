package docker

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

const repository = "aurora/oracle8"
const tag = "1"

const expected_version = "1.7.0"

func TestPullManifest(t *testing.T) {

	server, err := startMockRegistryServer("testdata/manifest.json")

	defer server.Close()

	if err != nil {
		t.Fatal(err)
	}

	target := NewRegistryClient(server.URL)

	manifest, err := target.GetManifest(repository, tag)

	if err != nil {
		t.Fatal(err)
	}

	if manifest.Name != repository {
		t.Errorf("Expected %s, got %s", repository, manifest.Name)
	}

	if manifest.Tag != tag {
		t.Errorf("Expected %s, got %s", tag, manifest.Tag)
	}
}

func TestGetManifestEnv(t *testing.T) {

	server, err := startMockRegistryServer("testdata/manifest.json")

	defer server.Close()

	if err != nil {
		t.Fatal(err)
	}

	target := NewRegistryClient(server.URL)

	actual_version, err := target.GetManifestEnv("aurora/oracle8", "1", "BASE_IMAGE_VERSION")

	if err != nil {
		t.Fatal(err)
	}

	if actual_version != expected_version {
		t.Errorf("Expected value %s, actual value is %s", expected_version, actual_version)
	}
}

func TestGetManifestEnvMap(t *testing.T) {

	server, err := startMockRegistryServer("testdata/manifest.json")

	defer server.Close()

	if err != nil {
		t.Fatal(err)
	}

	target := NewRegistryClient(server.URL)

	envMap, err := target.GetManifestEnvMap("aurora/oracle8", "1")

	if err != nil {
		t.Fatal(err)
	}

	expected_len := 14
	actual_len := len(envMap)

	if expected_len != actual_len {
		t.Errorf("Expected map size %d, actual size is %d", expected_len, actual_len)
	}
}

func TestGetTags(t *testing.T) {
	server, err := startMockRegistryServer("testdata/tags.list.json")

	defer server.Close()

	if err != nil {
		t.Fatal(err)
	}

	expectedTags := []string{"latest", "develop-SNAPSHOT",
		"develop-SNAPSHOT-9be2b9ca43a024415947a6c262e183406dbb090b",
		"2.0.0", "1.3.0", "1.2.1", "1.1.2", "1.1", "1.2", "1.3", "2.0", "2", "1"}

	target := NewRegistryClient(server.URL)

	tags, err := target.GetTags("aurora/oracle8")

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
