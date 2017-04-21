package docker

import (
	"io/ioutil"
	"testing"
	"net/http/httptest"
	"net/http"
	"fmt"
)

const expected_version = "1.7.0"

type RegistryClientMock struct {
}

func (registry *RegistryClientMock) PullManifest(repository string, tag string) ([]byte, error) {
	return testmanifest()
}

func TestGetManifestEnv(t *testing.T) {

	b, err := testmanifest()

	if err != nil {
		t.Fatalf("Failed to load test manifest; %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(b)))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(b)
	}))

	defer ts.Close()

	rc := NewRegistryClient(ts.URL)

	actual_version, err := GetManifestEnv(*rc, "aurora/oracle8", "1", "BASE_IMAGE_VERSION")

	if err != nil {
		t.Fatal(err)
	}

	if actual_version != expected_version {
		t.Errorf("Expected %s, got %s", expected_version, actual_version)
	}
}

func testmanifest() ([]byte, error) {
	buf, err := ioutil.ReadFile("testdata/manifest.json")

	if err != nil {
		return nil, err
	}

	return buf, nil
}
