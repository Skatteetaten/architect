package docker

import (
	"io"
	"io/ioutil"
	"encoding/json"

	"github.com/pkg/errors"
)

type DockerConfig struct {
	Auths Auths `json:"auths"`
	HttpHeaders map[string]string `json:"HttpHeaders,omitempty"`
}

type Auths map[string]RegistryEntry

type RegistryEntry struct {
	Email string `json:"email,omitempty"`
	Auth string `json:"auth,omitempty"`
}

func ReadConfig(reader io.Reader) (*DockerConfig, error) {
	var cfg *DockerConfig

	content, err := ioutil.ReadAll(reader)

	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(content, &cfg); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal Docker config json")
	}

	return cfg, nil
}
