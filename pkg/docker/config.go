package docker

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"strings"
)

type dockerConfig struct {
	Auths       auths             `json:"auths"`
	HTTPHeaders map[string]string `json:"httpHeaders,omitempty"`
}

type auths map[string]registryEntry

type registryEntry struct {
	Email string `json:"email,omitempty"`
	Auth  string `json:"auth,omitempty"`
}

type credentials struct {
	User     string
	Password string
}

func readConfig(reader io.Reader) (*dockerConfig, error) {
	var cfg *dockerConfig

	content, err := ioutil.ReadAll(reader)

	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(content, &cfg); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal Docker config json")
	}

	if len(cfg.Auths) > 0 {
		return cfg, nil
	}

	if err := json.Unmarshal(content, &cfg.Auths); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal Docker config using older json format")
	}

	return cfg, nil
}

func (cfg dockerConfig) getCredentials(address string) (*credentials, error) {
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

	return &credentials{User: strings.TrimSpace(creds[0]), Password: strings.TrimSpace(creds[1])}, nil
}
