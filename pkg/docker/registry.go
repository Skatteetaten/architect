package docker

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/docker/image"
	"io/ioutil"
	"net/http"
	"strings"
)

type ManifestProvider interface {
	GetManifest(repository string, tag string) (*schema1.SignedManifest, error)
	GetManifestEnv(repository string, tag string, name string) (string, error)
}

type TagsProvider interface {
	GetTags(repository string) (*tagsAPIResponse, error)
}

type RegistryClient struct {
	address string
}

func NewRegistryClient(address string) *RegistryClient {
	return &RegistryClient{address: address}
}

type tagsAPIResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func (registry *RegistryClient) GetManifest(repository string, tag string) (*schema1.SignedManifest, error) {
	url := fmt.Sprintf("%s/v2/%s/manifests/%s", registry.address, repository, tag)

	//TODO! Flytt alle HTTP metoder til felles utility-bibliotek!
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	res, err := client.Get(url)

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

func (registry *RegistryClient) GetTags(repository string) (*tagsAPIResponse, error) {
	url := fmt.Sprintf("%s/v2/%s/tags/list", registry.address, repository)
	var tagsList tagsAPIResponse
	res, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	err = json.Unmarshal(body, &tagsList)

	if err != nil {
		return nil, err
	}

	return &tagsList, nil
}

func (registry *RegistryClient) GetManifestEnv(repository string, tag string, name string) (string, error) {
	manifest, err := registry.GetManifest(repository, tag)

	if err != nil {
		return "", err
	}

	value, err := getManifestEnv(*manifest, name)

	if err != nil {
		return "", err
	}

	return value, nil
}

func getManifestEnv(manifest schema1.SignedManifest, name string) (string, error) {

	value, err := getEnvFromV1Data(manifest.History[0].V1Compatibility, name)

	if err != nil {
		return "", fmt.Errorf("Failed to extract env variable %s from manifest for image %s, tag %s: %v",
			name, manifest.Name, manifest.Tag, err)
	}

	return value, nil
}

func getEnvFromV1Data(v1data string, name string) (string, error) {
	var v1image image.V1Image

	if err := json.Unmarshal([]byte(v1data), &v1image); err != nil {
		return "", err
	}

	for _, entry := range v1image.Config.Env {
		key, value, err := envKeyValue(entry)

		if err != nil {
			continue
		}

		if key == name {
			return value, nil
		}
	}

	return "", nil
}

func envKeyValue(target string) (string, string, error) {
	s := strings.Split(target, "=")

	if len(s) != 2 {
		return "", "", fmt.Errorf("Invalid env declaration: %s", target)
	}

	return strings.TrimSpace(s[0]), strings.TrimSpace(s[1]), nil

}
