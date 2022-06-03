package docker_test

import (
	"github.com/skatteetaten/architect/v2/pkg/docker"
	"strings"
	"testing"
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

	if !ok {
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

	if !ok {
		t.Errorf("Failed to read valid json config")
	} else if auth.Auth != "au3rwp0kgisxz" {
		t.Errorf("Unexpected token %s", auth.Auth)
	}

}

func TestGetCredentials(t *testing.T) {
	// auth is "foo:barpw" base64 encoded
	config_json := `{
	"auths": {
		"the-registry": {
			"auth": "Zm9vOmJhcnB3Cg==",
			"email": "john.doe@foo.no"
		}
	}
}`

	expected_user := "foo"
	expected_password := "barpw"

	cfg, err := docker.ReadConfig(strings.NewReader(config_json))

	if err != nil || cfg == nil {
		t.Errorf("Failed to read valid json config: %v", err)
	}

	cred, err := cfg.GetCredentials("the-registry")

	if err != nil {
		t.Errorf("Failed to extract credentials: %v", err)
	}

	if cred.User != expected_user {
		t.Errorf("Expected user %s, actual was %s", expected_user, cred.User)
	}

	if cred.Password != expected_password {
		t.Errorf("Expected password %s, actual was %s", expected_password, cred.Password)
	}

}

func TestMultipleCredentialsInOldFormat(t *testing.T) {
	config_json := `{
		"the-registry": {
			"auth": "Zm9vOmJhcnB3Cg==",
			"email": "john.doe@foo.no"
		},
		"the-other-registry": {
			"auth": "bXk6cGFzcwo=",
			"email": "john.doe@foo.no"
		}
	}`

	cfg, err := docker.ReadConfig(strings.NewReader(config_json))

	if err != nil || cfg == nil {
		t.Errorf("Failed to read valid json config: %v", err)
	}

	expected_user := "foo"
	expected_password := "barpw"

	cred, err := cfg.GetCredentials("the-registry")

	if err != nil {
		t.Errorf("Failed to extract credentials: %v", err)
	}

	if cred.User != expected_user {
		t.Errorf("Expected user %s, actual was %s", expected_user, cred.User)
	}

	if cred.Password != expected_password {
		t.Errorf("Expected password %s, actual was %s", expected_password, cred.Password)
	}

	expected_user = "my"
	expected_password = "pass"

	cred, err = cfg.GetCredentials("the-other-registry")

	if err != nil {
		t.Errorf("Failed to extract credentials: %v", err)
	}

	if cred.User != expected_user {
		t.Errorf("Expected user %s, actual was %s", expected_user, cred.User)
	}

	if cred.Password != expected_password {
		t.Errorf("Expected password %s, actual was %s", expected_password, cred.Password)
	}

}
