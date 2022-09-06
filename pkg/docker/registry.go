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
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// Registry interface provides methods for interacting with a container registry
type Registry interface {
	GetImageInfo(ctx context.Context, repository string, tag string) (*runtime.ImageInfo, error)
	GetTags(ctx context.Context, repository string) (*TagsAPIResponse, error)
	GetImageConfig(ctx context.Context, repository string, digest string) (map[string]interface{}, error)
	GetManifest(ctx context.Context, repository string, digest string) (*ManifestV2, error)
	GetContainerConfig(ctx context.Context, repository string, digest string) (*ContainerConfig, error)
	LayerExists(ctx context.Context, repository string, layerDigest string) (bool, error)
	PushLayer(ctx context.Context, layer io.Reader, dstRepository string, layerDigest string) error
	PushManifest(ctx context.Context, manifest []byte, repository string, tag string) error
	PullLayer(ctx context.Context, repository string, layerDigest string) (string, error)
}

// Manifest schema representation
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

// RegistryConnectionInfo registry connection info
type RegistryConnectionInfo struct {
	Port        string
	Host        string
	Insecure    bool
	Credentials *RegistryCredentials
}

// RegistryClient configuration
type RegistryClient struct {
	connectionInfo RegistryConnectionInfo
	client         *http.Client
}

// BasicAuthWrapper RoundTrip
type BasicAuthWrapper struct {
	connectionInfo RegistryConnectionInfo
	next           http.RoundTripper
}

// RoundTrip append registry credentials
func (rt *BasicAuthWrapper) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	if rt.connectionInfo.Credentials != nil {
		req.SetBasicAuth(rt.connectionInfo.Credentials.Username, rt.connectionInfo.Credentials.Password)
	}
	return rt.next.RoundTrip(req)
}

// NewRegistryClient create new registry client
func NewRegistryClient(connectionInfo RegistryConnectionInfo) Registry {

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: connectionInfo.DisableTLSValidation()},
	}

	tr := BasicAuthWrapper{
		connectionInfo: connectionInfo,
		next:           transport,
	}

	client := &http.Client{Transport: &tr}
	return &RegistryClient{connectionInfo: connectionInfo, client: client}
}

// TagsAPIResponse list tags registry response
type TagsAPIResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

const (
	httpHeaderManifestSchemaV2 = "application/vnd.docker.distribution.manifest.v2+json"
	httpHeaderContainerImageV1 = "application/vnd.docker.container.image.v1+json"
)

// URL create registry url
func (r *RegistryConnectionInfo) URL() *url.URL {
	hostAndPort := fmt.Sprintf("%s:%s", r.Host, r.Port)
	logrus.Debugf("Host: %s", r.Host)
	u := url.URL{
		Scheme: "https",
		Host:   hostAndPort,
	}
	return &u
}

// DisableTLSValidation disable tls check
func (r *RegistryConnectionInfo) DisableTLSValidation() bool {
	return r.Insecure
}

func (registry *RegistryClient) getRegistryManifest(ctx context.Context, repository string, tag string) ([]byte, error) {
	mHeader := make(map[string]string)
	mHeader["Accept"] = httpHeaderManifestSchemaV2
	manifestURL := fmt.Sprintf("%s/v2/%s/manifests/%s", registry.connectionInfo.URL(), repository, tag)
	logrus.Infof("Retrieving registry manifest from URL %s", manifestURL)
	body, err := getHTTPRequest(ctx, registry.client, mHeader, manifestURL)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed in getRegistryManifest for request url %s with header %s", manifestURL, mHeader)
	}
	return body, nil
}

// GetManifest returns the image manifest
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

// GetContainerConfig returns the image's container configuration
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
	blobURL := fmt.Sprintf("%s/v2/%s/blobs/%s", registry.connectionInfo.URL(), repository, digestID)
	logrus.Debugf("Retrieving registry blob from URL %s", blobURL)
	body, err := getHTTPRequest(ctx, registry.client, mHeader, blobURL)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed in getRegistryBlob for request url %s and header %s", blobURL, mHeader)
	}
	return body, nil
}

// GetImageConfig get image config
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

// GetImageInfo get information about an image
func (registry *RegistryClient) GetImageInfo(ctx context.Context, repository string, tag string) (*runtime.ImageInfo, error) {
	body, err := registry.getRegistryManifest(ctx, repository, tag)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read manifest for repository %s, tag %s from Docker registry %s", repository, tag, registry.connectionInfo.URL())
	}

	manifestDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(body))

	manifestMeta := &Manifest{}
	err = json.Unmarshal(body, &manifestMeta)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to unmarshal manifest for repository %s, tag %s from Docker registry %s", repository, tag, registry.connectionInfo.URL())
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
			return nil, errors.Wrapf(err, "Failed to read image meta from blob in repository %s, digestID %s from Docker registry %s", repository, digestID, registry.connectionInfo.URL())
		}
		err = json.Unmarshal(body, &v1Image)

		if err != nil {
			return nil, errors.Wrapf(err, "Failed to unmarshal image meta from blob in repository %s, digestID %s from Docker registry %s", repository, digestID, registry.connectionInfo.URL())
		}

	} else {
		return nil, errors.Errorf("Error getting image manifest for %s from docker registry %s", repository, registry.connectionInfo.URL())
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
		Environment:              envMap,
		CompleteBaseImageVersion: baseImageVersion,
		Digest:                   manifestDigest,
	}, nil
}

// LayerExists checks if layer exists in registry
func (registry *RegistryClient) LayerExists(ctx context.Context, repository string, layerDigest string) (bool, error) {
	//HEAD /v2/<repository>/blobs/<digest>
	path := fmt.Sprintf("/v2/%s/blobs/%s", repository, layerDigest)
	logrus.Debugf("Check layer: %s", path)

	req, err := registry.newRequest(ctx, "HEAD", path, nil)
	if err != nil {
		return false, errors.Wrap(err, "LayerExists: Could not create request")
	}

	resp, err := registry.client.Do(req)
	if err != nil {
		return false, errors.Wrap(err, "LayerExists: Request failed")
	}

	if resp.StatusCode == 200 {
		return true, nil
	}
	logrus.Debugf("Could not find layer %s. Got response=%d", layerDigest, resp.StatusCode)

	return false, nil
}

// PullLayer pull image blob from registry
func (registry *RegistryClient) PullLayer(ctx context.Context, repository string, layerDigest string) (string, error) {

	path := fmt.Sprintf("/v2/%s/blobs/%s", repository, layerDigest)

	req, err := registry.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return "", errors.Wrap(err, "Could not create the blob download request")
	}

	resp, err := registry.client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "Failed download")
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected http code %d", resp.StatusCode)
	}

	file, err := os.CreateTemp("/tmp", "layer.*.tar.gz")
	if err != nil {
		return "", errors.Wrap(err, "Could not create temporary file")
	}

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "Could not write layer")
	}
	logrus.Infof("Pulled layer: %s", layerDigest)

	return file.Name(), err

}

// PushLayer push image blob
func (registry *RegistryClient) PushLayer(ctx context.Context, layer io.Reader, repository string, layerDigest string) error {
	//v2/repository/blobs/uploads/
	path := fmt.Sprintf("/v2/%s/blobs/uploads/", repository)

	//Start the transaction
	req, err := registry.newRequest(ctx, "POST", path, bytes.NewBufferString(""))
	if err != nil {
		return errors.Wrap(err, "Request creation failed")
	}

	resp, err := registry.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "PushLayer: Request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		return errors.Errorf("PushLayer start: Unexpected error code %d: From server: %s", resp.StatusCode, resp.Status)
	}

	location := resp.Header.Get("Location")
	req, err = registry.newRequest(ctx, "PATCH", location, layer)
	if err != nil {
		return errors.Wrap(err, "Upload request creation failed")
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err = registry.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "Layer upload filed")
	}

	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		respData, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "PushLayers patch: Unable to parse response body")
		}
		return errors.Errorf("PushLayer patch: Unexpected http code %d. From server=%s", resp.StatusCode, string(respData))
	}

	location = resp.Header.Get("Location")
	req, err = registry.newRequest(ctx, "PUT", location, nil)
	if err != nil {
		return errors.Wrap(err, "Commit request creation failed")
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	query := req.URL.Query()
	query.Add("digest", layerDigest)
	req.URL.RawQuery = query.Encode()

	resp, err = registry.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "The commit request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "Unable to read body")
		}
		bodyString := string(bodyBytes)
		return errors.Errorf("PushLayer commit: Got unexpected http response %d. From server: %s", resp.StatusCode, bodyString)
	}

	logrus.Infof("Pushed layer %s:%s", repository, layerDigest)
	return nil
}

// PushManifest push manifest
func (registry *RegistryClient) PushManifest(ctx context.Context, manifest []byte, repository string, tag string) error {
	//PUT /v2/<repository>/manifests/<tag>
	path := fmt.Sprintf("/v2/%s/manifests/%s", repository, tag)
	req, err := registry.newRequest(ctx, "PUT", path, bytes.NewBuffer(manifest))
	if err != nil {
		return errors.Wrap(err, "PushManifest: request creation failed")
	}

	req.Header.Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := registry.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "PushManifest: Request failed")
	}

	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		respData, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "PushManifest: ")
		}
		return errors.Errorf("PushManifest: Unexpected http code %d. From server: %s ", resp.StatusCode, string(respData))
	}

	return nil
}

// GetTags return image tags for a given repository
func (registry *RegistryClient) GetTags(ctx context.Context, repository string) (*TagsAPIResponse, error) {
	path := fmt.Sprintf("/v2/%s/tags/list", repository)
	var tagsList TagsAPIResponse

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: registry.connectionInfo.DisableTLSValidation()},
	}
	client := &http.Client{Transport: tr}

	req, err := registry.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "GetTags: Failed to create request")
	}

	resp, err := client.Do(req)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to download tags for repository %s", repository)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read tags for repository %s", repository)
	}

	err = json.Unmarshal(body, &tagsList)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to unmarshal tag list for repository %s ", repository)
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

func (registry *RegistryClient) newRequest(ctx context.Context, method string, path string, body io.Reader) (*http.Request, error) {

	reference, err := url.ParseRequestURI(path)
	if err != nil {
		return nil, errors.Wrap(err, "Request creation failed")
	}

	u := registry.connectionInfo.URL().ResolveReference(reference)

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, errors.Wrap(err, "Request creation failed")
	}
	return req, nil
}
