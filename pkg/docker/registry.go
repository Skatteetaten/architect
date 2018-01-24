package docker

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"github.com/docker/docker/image"
	"github.com/pkg/errors"
)

type ImageInfoProvider interface {
	GetCompleteBaseImageVersion(repository string, tag string) (string, error)
	GetTags(repository string) (*TagsAPIResponse, error)
	GetManifestEnvMap(repository string, tag string) (map[string]string, error)
}

type ContainerImageV1 struct {
	ContainerConfig struct {
		Env []string `json:"Env"`
	} `json:"config"`
}

type Manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"config"`
	History []struct {
		V1Compatibility string `json:"v1Compatibility"`
	}
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

const (
	httpHeaderManifestSchemaV2 = "application/vnd.docker.distribution.manifest.v2+json"
	httpHeaderContainerImageV1 = "application/vnd.docker.container.image.v1+json"
)

func (registry *RegistryClient) getRegistryManifest(repository string, tag string) ([]byte, error) {
	mHeader := make(map[string]string)
	mHeader["Accept"] = httpHeaderManifestSchemaV2
	url := fmt.Sprintf("%s/v2/%s/manifests/%s", registry.address, repository, tag)
	body, err := GetHTTPRequest(mHeader, url)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed in getRegistryManifest for request url %s and header %s", url, mHeader)
	}
	return body, nil
}

func (registry *RegistryClient) getRegistryBlob(repository string, digestID string) ([]byte, error) {
	mHeader := make(map[string]string)
	mHeader["Accept"] = httpHeaderContainerImageV1
	url := fmt.Sprintf("%s/v2/%s/blobs/%s", registry.address, repository, digestID)
	body, err := GetHTTPRequest(mHeader, url)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed in getRegistryBlob for request url %s and header %s", url, mHeader)
	}
	return body, nil
}

func (registry *RegistryClient) GetManifestEnvMap(repository string, tag string) (map[string]string, error) {
	body, err := registry.getRegistryManifest(repository, tag)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read manifest for repository %s, tag %s from Docker registry %s", repository, tag, registry.address)
	}

	manifestMeta := &Manifest{}
	err = json.Unmarshal(body, &manifestMeta)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to unmarshal manifest for repository %s, tag %s from Docker registry %s", repository, tag, registry.address)
	}

	if manifestMeta.SchemaVersion == 1 {
		if len(manifestMeta.History) > 0 {
			envMap, err := getEnvMapFromV1Data(manifestMeta.History[0].V1Compatibility)

			if err != nil {
				return nil, errors.Wrap(err, "Unable to get environment map for schemaVersion 1")
			}

			return envMap, nil
		}
		return nil, errors.New("Error in Manifest for schemaVersion 1. Incomplete History list")
	} else if manifestMeta.Config.Digest != "" {
		digestID := manifestMeta.Config.Digest

		body, err = registry.getRegistryBlob(repository, digestID)

		if err != nil {
			return nil, errors.Wrapf(err, "Failed to read image meta from blob in repository %s, digestID %s from Docker registry %s", repository, digestID, registry.address)
		}

		imageMeta := ContainerImageV1{}
		err = json.Unmarshal(body, &imageMeta)

		if err != nil {
			return nil, errors.Wrapf(err, "Failed to unmarshal image meta from blob in repository %s, digestID %s from Docker registry %s", repository, digestID, registry.address)
		}

		envMap := make(map[string]string)
		for _, entry := range imageMeta.ContainerConfig.Env {
			key, value, err := envKeyValue(entry)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to read env variable")
			}
			envMap[key] = value
		}

		return envMap, nil
	}
	return nil, errors.Errorf("Failed creating environment map from manifest in registry %s and repository %s", registry.address, repository)
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
