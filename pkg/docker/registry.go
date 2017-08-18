package docker

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/docker/image"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"strings"
)

type ImageInfoProvider interface {
	GetCompleteBaseImageVersion(repository string, tag string) (string, error)
	GetTags(repository string) (*TagsAPIResponse, error)
	GetManifestEnvMap(repository string, tag string) (map[string]string, error)
}

type RegistryClient struct {
	address string
}

func NewRegistryClient(address string) ImageInfoProvider {
	return &RegistryClient{address: address}
}

type TagsAPIResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func (registry *RegistryClient) getManifest(repository string, tag string) (*schema1.SignedManifest, error) {
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
		return nil, errors.Wrapf(err, "Failed to unmarshal manifest for repository %s, tag %s from Docker registry %s", repository, tag, url)
	}

	return manifest, nil
}

func (registry *RegistryClient) GetTags(repository string) (*TagsAPIResponse, error) {
	url := fmt.Sprintf("%s/v2/%s/tags/list", registry.address, repository)
	var tagsList TagsAPIResponse

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	res, err := client.Get(url)

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
		return nil, errors.Wrapf(err, "Failed to unmarshal tag list for repository %s from Docker registry %s", repository, url)
	}

	return &tagsList, nil
}

func (registry *RegistryClient) GetManifestEnvMap(repository string, tag string) (map[string]string, error) {
	manifest, err := registry.getManifest(repository, tag)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to get manifest")
	}

	return getEnvMapFromV1Data(manifest.History[0].V1Compatibility)
}

func (registry *RegistryClient) GetCompleteBaseImageVersion(repository string, tag string) (string, error) {

	envMap, err := registry.GetManifestEnvMap(repository, tag)

	if err != nil {
		return "", errors.Wrap(err, "Unable to get environment map")
	}

	value, ok := envMap["BASE_IMAGE_VERSION"]

	if !ok {
		return "", errors.Errorf("Env variable %s not in manifest", "BASE_IMAGE_VERSION")
	}

	if value == "" {
		return "", errors.Errorf("Failed to extract version in getBaseImageVersion, registry: %s, "+
			"BaseImage: %s, BaseVersion: %s EnvMap: %v",
			registry, repository, tag, envMap)
	}
	return value, nil
}

func getEnvMapFromV1Data(v1data string) (map[string]string, error) {
	var v1image image.V1Image

	envMap := make(map[string]string)

	if err := json.Unmarshal([]byte(v1data), &v1image); err != nil {
		return nil, errors.Wrapf(err, "Failed to unmarshal image from manifest")
	}

	for _, entry := range v1image.Config.Env {
		key, value, err := envKeyValue(entry)

		if err != nil {
			return nil, errors.Wrap(err, "Failed to read env variable")
		}

		envMap[key] = value
	}

	return envMap, nil
}

func envKeyValue(target string) (string, string, error) {
	s := strings.Split(target, "=")

	if len(s) != 2 {
		return "", "", errors.Errorf("Invalid env declaration: %s", target)
	}

	return strings.TrimSpace(s[0]), strings.TrimSpace(s[1]), nil

}
