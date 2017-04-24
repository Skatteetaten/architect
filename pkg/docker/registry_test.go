package docker

import (
	"testing"
)

const repository = "aurora/oracle8"
const tag = "1"

func TestPullManifest(t *testing.T) {

	server, err := StartMockRegistry()

	defer server.Close()

	if err != nil {
		t.Fatal(err)
	}

	target := NewRegistryClient(server.URL)

	manifest, err := target.PullManifest( repository, tag)

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

