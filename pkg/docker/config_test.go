package docker_test

import (
	"testing"
	"github.com/skatteetaten/architect/pkg/docker"
	"strings"
)

func TestReadConfigSingleRegistry(t *testing.T) {

	config_json := `{
	"auths": {
		"foo-registry": {
			"auth": "b3rb3xc0was7=",
			"email": "john.doe@foo.no"
		}
	}
}`

	cfg, err := docker.ReadConfig(strings.NewReader(config_json))

	if err != nil || cfg == nil {
		t.Errorf("Failed to read valid json config: %v", err)
	}

	auth, ok := cfg.Auths["foo-registry"]

	if ! ok {
		t.Errorf("Failed to read valid json config")
	} else if auth.Auth != "b3rb3xc0was7=" {
		t.Errorf("Unexpected token %s", auth.Auth)
	}
}

func TestReadConfigMultipleRegistries(t *testing.T) {

	config_json := `{
	"auths": {
		"foo-registry": {
			"auth": "b3rb3xc0was7="
		},
		"bar-registry": {
			"auth": "au3rwp0kgisxz"
		},
		"baz-registry": {
			"auth": "g0f5ev3cadb=="
		}
	}
}`

	cfg, err := docker.ReadConfig(strings.NewReader(config_json))

	if err != nil || cfg == nil {
		t.Errorf("Failed to read valid json config: %v", err)
	}

	auth, ok := cfg.Auths["bar-registry"]

	if ! ok {
		t.Errorf("Failed to read valid json config")
	} else if auth.Auth != "au3rwp0kgisxz" {
		t.Errorf("Unexpected token %s", auth.Auth)
	}

}