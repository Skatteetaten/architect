package docker

import (
	"io/ioutil"
	"testing"
)

const expected_value = "1.7.0"

type RegistryClientMock struct {
}

func (registry *RegistryClientMock) PullManifest(repository string, tag string) ([]byte, error) {
	return testmanifest()
}

func TestGetManifestEnv(t *testing.T) {

	// Uncomment to pull manifest from registry:
	//rc := NewHttpClient("http://uil0map-paas-app01:9090")

	rc := &RegistryClientMock{}

	actual, err := GetManifestEnv(rc, "aurora/oracle8", "1", "BASE_IMAGE_VERSION")

	if err != nil {
		t.Error(err)
	}

	if actual != expected_value {
		t.Error("Expected", expected_value, "got", actual)
	}
}

func testmanifest() ([]byte, error) {
	buf, err := ioutil.ReadFile("testdata/manifest.json")

	if err != nil {
		return nil, err
	}

	return buf, nil
}
