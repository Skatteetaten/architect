package docker

import (
	"fmt"
	"github.com/docker/distribution/manifest/schema1"
	"io/ioutil"
	"net/http"
)

type RegistryClient struct {
	address string
}

func NewRegistryClient(address string) *RegistryClient {
	return &RegistryClient{address: address}
}

func (registry *RegistryClient) PullManifest(repository string, tag string) (*schema1.SignedManifest, error) {
	url := fmt.Sprintf("%s/v2/%s/manifests/%s", registry.address, repository, tag)

	res, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	manifest := &schema1.SignedManifest{}

	if err = manifest.UnmarshalJSON(body); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal manifest for image %s, tag %s from Docker registry: %v", repository, tag, err)
	}

	return manifest, nil
}
