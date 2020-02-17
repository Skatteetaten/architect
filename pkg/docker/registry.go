package docker

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/image"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

type ImageInfoProvider interface {
	GetImageInfo(repository string, tag string) (*runtime.ImageInfo, error)
	GetTags(repository string) (*TagsAPIResponse, error)
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
	logrus.Debugf("Retrieving registry manifest from URL %s", url)
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
	logrus.Debugf("Retrieving registry blob from URL %s", url)
	body, err := GetHTTPRequest(mHeader, url)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed in getRegistryBlob for request url %s and header %s", url, mHeader)
	}
	return body, nil
}

func (registry *RegistryClient) GetImageInfo(repository string, tag string) (*runtime.ImageInfo, error) {
	body, err := registry.getRegistryManifest(repository, tag)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read manifest for repository %s, tag %s from Docker registry %s", repository, tag, registry.address)
	}

	manifestMeta := &Manifest{}
	err = json.Unmarshal(body, &manifestMeta)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to unmarshal manifest for repository %s, tag %s from Docker registry %s", repository, tag, registry.address)
	}

	var v1Image image.V1Image
	if manifestMeta.SchemaVersion == 1 {
		if len(manifestMeta.History) > 0 {
			if err := json.Unmarshal([]byte(manifestMeta.History[0].V1Compatibility), &v1Image); err != nil {
				return nil, errors.Wrapf(err, "Failed to unmarshal image from manifest")
			}
		}
	} else if manifestMeta.Config.Digest != "" {
		digestID := manifestMeta.Config.Digest

		body, err = registry.getRegistryBlob(repository, digestID)

		if err != nil {
			return nil, errors.Wrapf(err, "Failed to read image meta from blob in repository %s, digestID %s from Docker registry %s", repository, digestID, registry.address)
		}
		err = json.Unmarshal(body, &v1Image)

		if err != nil {
			return nil, errors.Wrapf(err, "Failed to unmarshal image meta from blob in repository %s, digestID %s from Docker registry %s", repository, digestID, registry.address)
		}

	} else {
		return nil, errors.Errorf("Error getting image manifest for %s from docker registry %s", repository, registry.address)
	}

	envMap := make(map[string]string)
	for _, entry := range v1Image.Config.Env {
		key, value, err := envKeyValue(entry)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to read env variable")
		}
		envMap[key] = value
	}

	baseImageVersion, exists := envMap["BASE_IMAGE_VERSION"]
	if !exists {
		return nil, errors.Errorf("Unable to get BASE_IMAGE_VERSION. %s is not a compatible image", repository)
	}

	return &runtime.ImageInfo{
		Labels:                   v1Image.Config.Labels,
		Enviroment:               envMap,
		CompleteBaseImageVersion: baseImageVersion,
	}, nil
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

	tagsList.Tags = ConvertRepositoryTagsToTags(tagsList.Tags)

	return &tagsList, nil
}

func envKeyValue(target string) (string, string, error) {
	regex := regexp.MustCompile("(.*?)=(.*)")
	if regex.MatchString(target) {
		matches := regex.FindStringSubmatch(target)
		return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2]), nil
	}
	return "", "", errors.Errorf("Invalid env declaration: %s", target)
}
