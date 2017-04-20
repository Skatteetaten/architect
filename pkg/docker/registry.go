package docker

import (
	"encoding/json"
	"fmt"
	"github.com/docker/distribution/manifest/schema1"
	"io/ioutil"
	"net/http"
	"strings"
)

type RegistryClient interface {
	PullManifest(repository string, tag string) ([]byte, error)
}

type HttpClient struct {
	address string
}

func NewHttpClient(address string) *HttpClient {
	return &HttpClient{address: address}
}

func (registry *HttpClient) PullManifest(repository string, tag string) ([]byte, error) {
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

	return body, nil
}

func GetManifestEnv(client RegistryClient, repository string, tag string, name string) (string, error) {

	body, err := client.PullManifest(repository, tag)

	if err != nil {
		return "", err
	}

	manifest := &schema1.SignedManifest{}

	if err = manifest.UnmarshalJSON(body); err != nil {
		return "", fmt.Errorf("Failed to unmarshal manifest for image %s, tag %s from Docker registry: %v", repository, tag, err)
	}

	if err != nil {
		return "", err
	}

	value, err := getEnvFromV1Data(manifest.History[0].V1Compatibility, name)

	if err != nil {
		return "", fmt.Errorf("Failed to extract env variable %s from manifest for image %s, tag %s: %v", name, repository, tag, err)
	}

	return value, nil
}

func getEnvFromV1Data(v1data string, name string) (string, error) {

	m := map[string]interface{}{}

	if err := json.Unmarshal([]byte(v1data), &m); err != nil {
		return "", err
	}

	configMap, err := getConfigMap(m)

	if err != nil {
		return "", err
	}

	envArray, err := getEnvArray(configMap)

	if err != nil {
		return "", err
	}

	for _, entry := range envArray {
		dec, ok := entry.(string)

		if !ok {
			return "", fmt.Errorf("Failed to read variable")
		}

		key, value, err := envKeyValue(dec)

		if err != nil {
			continue
		}

		if key == name {
			return value, nil
		}
	}

	return "", nil
}

func getConfigMap(m map[string]interface{}) (map[string]interface{}, error) {
	if m["config"] == nil {
		return nil, fmt.Errorf("Missing \"config\" object")
	}

	config, ok := m["config"].(map[string]interface{})

	if !ok {
		return nil, fmt.Errorf("Object \"config\" has unexpected type")
	}

	return config, nil
}

func getEnvArray(m map[string]interface{}) ([]interface{}, error) {
	if m["Env"] == nil {
		return nil, fmt.Errorf("Missing \"Env\" object")
	}

	env, ok := m["Env"].([]interface{})

	if !ok {
		return nil, fmt.Errorf("Object \"Env\" has unexpected type")
	}

	return env, nil
}

func envKeyValue(target string) (string, string, error){
	s := strings.Split(target, "=")

	if len(s) != 2 {
		return "", "", fmt.Errorf("Invalid env declaration: %s", target)
	}

	return strings.TrimSpace(s[0]), strings.TrimSpace(s[1]), nil

}
