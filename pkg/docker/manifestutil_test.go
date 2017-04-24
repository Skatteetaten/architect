package docker

import (
	"testing"
	"io/ioutil"
	"github.com/docker/distribution/manifest/schema1"
)

const expected_version = "1.7.0"

func TestGetManifestEnv(t *testing.T) {

	manifest, err := testManifest()

	if err != nil {
		t.Fatal(err)
	}

	actual_version, err := GetManifestEnv(*manifest, "BASE_IMAGE_VERSION")

	if err != nil {
		t.Fatal(err)
	}

	if actual_version != expected_version {
		t.Errorf("Expected %s, got %s", expected_version, actual_version)
	}
}

func testManifest() (*schema1.SignedManifest, error) {
	buf, err := ioutil.ReadFile("testdata/manifest.json")

	if err != nil {
		return nil, err
	}

	manifest := &schema1.SignedManifest{}

	if err = manifest.UnmarshalJSON(buf); err != nil {
		return nil, err
	}

	return manifest, nil
}