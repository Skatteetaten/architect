package docker

import (
	"fmt"
	"encoding/json"
	"strings"
	"github.com/docker/docker/image"
	"github.com/docker/distribution/manifest/schema1"
)

func GetManifestEnv(manifest schema1.SignedManifest, name string) (string, error) {

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

