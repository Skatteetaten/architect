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
	"github.com/pkg/errors"
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
		return nil, errors.Wrapf(err, "Failed to download manifest for repository %s, tag %s from Docker registry %s", repository, tag, url)
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read manifest for repository %s, tag %s from Docker registry %s", repository, tag, url)
	}

	defer res.Body.Close()

	manifest := &schema1.SignedManifest{}

	if err = manifest.UnmarshalJSON(body); err != nil {
		return nil, errors.Wrapf(err,"Failed to unmarshal manifest for repository %s, tag %s from Docker registry %s", repository, tag, url)
	}

	return manifest, nil
}

func (registry *RegistryClient) GetTags(repository string) (*tagsAPIResponse, error) {
	url := fmt.Sprintf("%s/v2/%s/tags/list", registry.address, repository)
	var tagsList tagsAPIResponse
	res, err := http.Get(url)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to download tags for repository %s from Docker registry %s", repository, url)
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read tags for repository %s from Docker registry %s", repository, url)
	}

	defer res.Body.Close()

	err = json.Unmarshal(body, &tagsList)

	if err != nil {
		return nil, errors.Wrapf(err,"Failed to unmarshal tag list for repository %s from Docker registry %s", repository, url)
	}

	return &tagsList, nil
}

func (registry *RegistryClient) GetManifestEnv(repository string, tag string, name string) (string, error) {
	manifest, err := registry.GetManifest(repository, tag)

	if err != nil {
		return "", errors.Wrap(err, "Failed to get manifest")
	}

	value, err := getManifestEnv(*manifest, name)

	if err != nil {
		return "", errors.Wrapf(err, "Failed to parse manifest for repository %s, tag %s", repository, tag)
	}

	return value, nil
}

func getManifestEnv(manifest schema1.SignedManifest, name string) (string, error) {

	value, err := getEnvFromV1Data(manifest.History[0].V1Compatibility, name)

	if err != nil {
		return "", errors.Wrapf(err, "Failed to extract env variable %s from manifest", name)
	}

	return value, nil
}

func getEnvFromV1Data(v1data string, name string) (string, error) {
	var v1image image.V1Image

	if err := json.Unmarshal([]byte(v1data), &v1image); err != nil {
		return "", errors.Wrapf(err, "Failed to unmarshal image from manifest")
	}

	for _, entry := range v1image.Config.Env {
		key, value, err := envKeyValue(entry)

		if err != nil {
			return "", errors.Wrap(err, "Failed to read env variable")
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
		return "", "", errors.Errorf("Invalid env declaration: %s", target)
	}

	return strings.TrimSpace(s[0]), strings.TrimSpace(s[1]), nil

}
