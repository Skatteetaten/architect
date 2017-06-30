package docker

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"strings"
)

type DockerConfig struct {
	Auths       Auths             `json:"auths"`
	HttpHeaders map[string]string `json:"HttpHeaders,omitempty"`
}

type Auths map[string]RegistryEntry

type RegistryEntry struct {
	Email string `json:"email,omitempty"`
	Auth  string `json:"auth,omitempty"`
}

type Credentials struct {
	User     string
	Password string
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

func (cfg DockerConfig) GetCredentials(address string) (*Credentials, error) {
	regEntry, ok := cfg.Auths[address]

	if !ok || regEntry.Auth == "" {
		return nil, nil
	}

	auth, err := base64.StdEncoding.DecodeString(regEntry.Auth)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to base64 decode credentials from Docker config for server %s", address)
	}

	creds := strings.Split(string(auth), ":")

	if len(creds) != 2 {
		return nil, errors.Errorf("Failed to extract username and password from Docker config for server %s", address)
	}

	return &Credentials{User: strings.TrimSpace(creds[0]), Password: strings.TrimSpace(creds[1])}, nil
}
