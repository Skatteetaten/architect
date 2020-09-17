package docker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type Registry interface {
	GetImageInfo(ctx context.Context, repository string, tag string) (*runtime.ImageInfo, error)
	GetTags(ctx context.Context, repository string) (*TagsAPIResponse, error)
	GetImageConfig(ctx context.Context, repository string, digest string) (map[string]interface{}, error)
	GetManifest(ctx context.Context, repository string, digest string) (*ManifestV2, error)
	GetContainerConfig(ctx context.Context, repository string, digest string) (*ContainerConfig, error)
	LayerExists(ctx context.Context, repository string, layerDigest string) (bool, error)
	MountLayer(ctx context.Context, srcRepository string, dstRepository string, layerDigest string) error
	PushLayer(ctx context.Context, file string, dstRepository string, layerDigest string) error
	PushManifest(ctx context.Context, file string, repository string, tag string) error
	PullLayer(ctx context.Context, repository string, layerDigest string) (string, error)
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
	pullRegistry string
	pushRegistry string
	credentials  *RegistryCredentials
}

//NewRegistryClient create new registry client
func NewRegistryClient(pullRegistry string, pushRegistry string, credentials *RegistryCredentials) Registry {
	return &RegistryClient{pullRegistry: pullRegistry, pushRegistry: pushRegistry, credentials: credentials}
}

type TagsAPIResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

const (
	httpHeaderManifestSchemaV2 = "application/vnd.docker.distribution.manifest.v2+json"
	httpHeaderContainerImageV1 = "application/vnd.docker.container.image.v1+json"
)

//TODO: Improve error handling

func (registry *RegistryClient) getRegistryManifest(ctx context.Context, repository string, tag string) ([]byte, error) {
	mHeader := make(map[string]string)
	mHeader["Accept"] = httpHeaderManifestSchemaV2
	url := fmt.Sprintf("%s/v2/%s/manifests/%s", registry.pullRegistry, repository, tag)
	logrus.Infof("Retrieving registry manifest from URL %s", url)
	body, err := GetHTTPRequest(ctx, mHeader, url)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed in getRegistryManifest for request url %s and header %s", url, mHeader)
	}
	return body, nil
}

func (registry *RegistryClient) GetManifest(ctx context.Context, repository string, tag string) (*ManifestV2, error) {

	data, err := registry.getRegistryManifest(ctx, repository, tag)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to fetch image manifest for image %s:%s", repository, tag)
	}

	var manifest ManifestV2
	err = json.Unmarshal(data, &manifest)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal of manifest failed")
	}

	return &manifest, nil
}

func (registry *RegistryClient) GetContainerConfig(ctx context.Context, repository string, digest string) (*ContainerConfig, error) {
	data, err := registry.getRegistryBlob(ctx, repository, digest)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to fetch container config blog for image %s", repository)
	}

	var config ContainerConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal of container config failed")
	}

	return &config, nil
}

func (registry *RegistryClient) getRegistryBlob(ctx context.Context, repository string, digestID string) ([]byte, error) {
	mHeader := make(map[string]string)
	mHeader["Accept"] = httpHeaderContainerImageV1
	url := fmt.Sprintf("%s/v2/%s/blobs/%s", registry.pullRegistry, repository, digestID)
	logrus.Debugf("Retrieving registry blob from URL %s", url)
	body, err := GetHTTPRequest(ctx, mHeader, url)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed in getRegistryBlob for request url %s and header %s", url, mHeader)
	}
	return body, nil
}

//GetImageConfig get image config
func (registry *RegistryClient) GetImageConfig(ctx context.Context, repository string, digest string) (map[string]interface{}, error) {
	var result map[string]interface{}

	manifest, err := registry.getRegistryManifest(ctx, repository, digest)
	if err != nil {
		return nil, err
	}

	manifestStruct := &Manifest{}
	err = json.Unmarshal(manifest, &manifestStruct)
	if err != nil {
		return nil, err
	}

	data, err := registry.getRegistryBlob(ctx, repository, manifestStruct.Config.Digest)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

//GetImageInfo get information about an image
func (registry *RegistryClient) GetImageInfo(ctx context.Context, repository string, tag string) (*runtime.ImageInfo, error) {
	body, err := registry.getRegistryManifest(ctx, repository, tag)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read manifest for repository %s, tag %s from Docker registry %s", repository, tag, registry.pullRegistry)
	}

	manifestDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(body))

	manifestMeta := &Manifest{}
	err = json.Unmarshal(body, &manifestMeta)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to unmarshal manifest for repository %s, tag %s from Docker registry %s", repository, tag, registry.pullRegistry)
	}

	var v1Image V1Image
	if manifestMeta.SchemaVersion == 1 {
		if len(manifestMeta.History) > 0 {
			if err := json.Unmarshal([]byte(manifestMeta.History[0].V1Compatibility), &v1Image); err != nil {
				return nil, errors.Wrapf(err, "Failed to unmarshal image from manifest")
			}
		}
	} else if manifestMeta.Config.Digest != "" {
		digestID := manifestMeta.Config.Digest

		body, err = registry.getRegistryBlob(ctx, repository, digestID)

		if err != nil {
			return nil, errors.Wrapf(err, "Failed to read image meta from blob in repository %s, digestID %s from Docker registry %s", repository, digestID, registry.pullRegistry)
		}
		err = json.Unmarshal(body, &v1Image)

		if err != nil {
			return nil, errors.Wrapf(err, "Failed to unmarshal image meta from blob in repository %s, digestID %s from Docker registry %s", repository, digestID, registry.pullRegistry)
		}

	} else {
		return nil, errors.Errorf("Error getting image manifest for %s from docker registry %s", repository, registry.pullRegistry)
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
		Digest:                   manifestDigest,
	}, nil
}

//LayerExists checks if layer exists in registry
func (registry *RegistryClient) LayerExists(ctx context.Context, repository string, layerDigest string) (bool, error) {
	//HEAD /v2/<repository>/blobs/<digest>
	url := fmt.Sprintf("%s/v2/%s/blobs/%s", registry.pushRegistry, repository, layerDigest)
	logrus.Debugf("Check layer: %s", url)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false, errors.Wrap(err, "LayerExists: Could not create request")
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, errors.Wrap(err, "LayerExists: Request failed")
	}

	if resp.StatusCode == 200 {
		return true, nil
	}
	logrus.Warnf("Could not find layer %s. Got response=%d", layerDigest, resp.StatusCode)

	return false, nil
}

func (registry *RegistryClient) PullLayer(ctx context.Context, repository string, layerDigest string) (string, error) {
	url := fmt.Sprintf(fmt.Sprintf("%s/v2/%s/blobs/%s", registry.pullRegistry, repository, layerDigest))
	logrus.Infof("Pull layer: %s", url)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", errors.Wrap(err, "Could not create the blob download request")
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "Failed download")
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected http code %d", resp.StatusCode)
	}

	file, err := ioutil.TempFile("/tmp", "layer")
	if err != nil {
		return "", errors.Wrap(err, "Could not create a temp file")
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "Could not copy content")
	}
	logrus.Infof("Pulled layer: %s", layerDigest)

	return file.Name(), err
}

//MountLayer performs cross mounting
//TODO: Insert layer instead off layer digest
func (registry *RegistryClient) MountLayer(ctx context.Context, srcRepository string, dstRepository string, layerDigest string) error {
	//"https://<address>/v2/<srcRepository>/blobs/uploads/?mount=<digest>&from=<dstRepository>"
	url := fmt.Sprintf(fmt.Sprintf("%s/v2/%s/blobs/uploads/?mount=%s&from=%s", registry.pushRegistry, dstRepository, layerDigest, srcRepository))
	logrus.Infof("Mount layer: %s", url)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return errors.Wrap(err, "Could not create the mount request")
	}
	if registry.credentials != nil {
		req.SetBasicAuth(registry.credentials.Username, registry.credentials.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "Failed to mount: src=%s dst=%s layerDigest=%s", srcRepository, dstRepository, layerDigest)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 && resp.StatusCode != 202 {
		respData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "MountLayer: Unable to read response body")
		}

		return fmt.Errorf("MountLayer: Request failed with status code = %d. From server: %s", resp.StatusCode, string(respData))
	}
	return nil
}

//PushLayer push layer
func (registry *RegistryClient) PushLayer(ctx context.Context, file string, repository string, layerDigest string) error {

	layer, err := os.Open(file)
	if err != nil {
		return errors.Wrapf(err, "Unable to open file %s", file)
	}

	//registry/v2/repository/blobs/uploads/
	url := fmt.Sprintf("%s/v2/%s/blobs/uploads/", registry.pushRegistry, repository)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	//Start the transaction
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(""))
	if err != nil {
		return errors.Wrap(err, "Request creation failed")
	}
	if registry.credentials != nil {
		req.SetBasicAuth(registry.credentials.Username, registry.credentials.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "PushLayer: Request failed")
	}

	if resp.StatusCode != 202 {
		respData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "Unable to read body")
		}

		return errors.Errorf("PushLayer start: Unexpected error code %d: From server: %s", resp.StatusCode, string(respData))
	}

	//Nexus and docker handles things a bit different in regard to the location header
	location := resp.Header.Get("Location")
	if !strings.Contains(location, registry.pushRegistry) {
		location = fmt.Sprintf("%s%s", registry.pushRegistry, location)
	}

	req, err = http.NewRequestWithContext(ctx, "PATCH", location, layer)
	if err != nil {
		return errors.Wrap(err, "Upload request creation failed")
	}
	if registry.credentials != nil {
		req.SetBasicAuth(registry.credentials.Username, registry.credentials.Password)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err = client.Do(req)
	if err != nil {
		return errors.Wrap(err, "Layer upload filed")
	}

	if resp.StatusCode != 202 {
		respData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "PushLayers patch: Unable to parse response body")
		}
		return errors.Errorf("PushLayer patch: Unexpected http code %d. From server=%s", resp.StatusCode, string(respData))
	}

	location = resp.Header.Get("Location")
	if !strings.Contains(location, registry.pushRegistry) {
		location = fmt.Sprintf("%s%s", registry.pushRegistry, location)
	}

	req, err = http.NewRequestWithContext(ctx, "PUT", location, nil)
	if err != nil {
		return errors.Wrap(err, "Commit request creation failed")
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	query := req.URL.Query()
	query.Add("digest", layerDigest)
	req.URL.RawQuery = query.Encode()

	if registry.credentials != nil {
		req.SetBasicAuth(registry.credentials.Username, registry.credentials.Password)
	}

	resp, err = client.Do(req)
	if err != nil {
		return errors.Wrap(err, "The commit request failed")
	}

	if resp.StatusCode != 201 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "Unable to read body")
		}
		bodyString := string(bodyBytes)
		return errors.Errorf("PushLayer commit: Got unexpected http response %d. From server: %s", resp.StatusCode, bodyString)
	}

	logrus.Infof("Pushed layer %s:%s", repository, layerDigest)
	return nil
}

//PushManifest push manifest
func (registry *RegistryClient) PushManifest(ctx context.Context, file string, repository string, tag string) error {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	manifest, err := os.Open(file)
	if err != nil {
		return errors.Wrapf(err, "Unable to open file %s", file)
	}

	//PUT /v2/<repository>/manifests/<tag>
	url := fmt.Sprintf("%s/v2/%s/manifests/%s", registry.pushRegistry, repository, tag)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, manifest)
	if err != nil {
		return errors.Wrap(err, "PushManifest: request creation failed")
	}
	if registry.credentials != nil {
		req.SetBasicAuth(registry.credentials.Username, registry.credentials.Password)
	}

	req.Header.Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "PushManifest: Request failed")
	}

	if resp.StatusCode != 201 {
		respData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "PushManifest: ")
		}
		return errors.Errorf("PushManifest: Unexpected http code %d. From server: %s ", resp.StatusCode, string(respData))
	}

	return nil
}

func (registry *RegistryClient) GetTags(ctx context.Context, repository string) (*TagsAPIResponse, error) {
	url := fmt.Sprintf("%s/v2/%s/tags/list", registry.pullRegistry, repository)
	var tagsList TagsAPIResponse

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "GetTags: Failed to create request")
	}

	res, err := client.Do(req)

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
