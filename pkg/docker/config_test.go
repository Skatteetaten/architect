package docker

import (
	"strings"
	"testing"
)

func TestReadConfigSingleRegistry(t *testing.T) {

	configJSON := `{
	"auths": {
		"foo-registry": {
			"auth": "b3rb3xc0was7=",
			"email": "john.doe@foo.no"
		}
	}
}`

	cfg, err := readConfig(strings.NewReader(configJSON))

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

	configJSON := `{
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

	cfg, err := readConfig(strings.NewReader(configJSON))

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
	configJSON := `{
	"auths": {
		"the-registry": {
			"auth": "Zm9vOmJhcnB3Cg==",
			"email": "john.doe@foo.no"
		}
	}
}`

	expectedUser := "foo"
	expectedPassword := "barpw"

	cfg, err := readConfig(strings.NewReader(configJSON))

	if err != nil || cfg == nil {
		t.Errorf("Failed to read valid json config: %v", err)
	}

	cred, err := cfg.getCredentials("the-registry")

	if err != nil {
		t.Errorf("Failed to extract credentials: %v", err)
	}

	if cred.User != expectedUser {
		t.Errorf("Expected user %s, actual was %s", expectedUser, cred.User)
	}

	if cred.Password != expectedPassword {
		t.Errorf("Expected password %s, actual was %s", expectedPassword, cred.Password)
	}

}

func TestMultipleCredentialsInOldFormat(t *testing.T) {
	configJSON := `{
		"the-registry": {
			"auth": "Zm9vOmJhcnB3Cg==",
			"email": "john.doe@foo.no"
		},
		"the-other-registry": {
			"auth": "bXk6cGFzcwo=",
			"email": "john.doe@foo.no"
		}
	}`

	cfg, err := readConfig(strings.NewReader(configJSON))

	if err != nil || cfg == nil {
		t.Errorf("Failed to read valid json config: %v", err)
	}

	expectedUser := "foo"
	expectedPassword := "barpw"

	cred, err := cfg.getCredentials("the-registry")

	if err != nil {
		t.Errorf("Failed to extract credentials: %v", err)
	}

	if cred.User != expectedUser {
		t.Errorf("Expected user %s, actual was %s", expectedUser, cred.User)
	}

	if cred.Password != expectedPassword {
		t.Errorf("Expected password %s, actual was %s", expectedPassword, cred.Password)
	}

	expectedUser = "my"
	expectedPassword = "pass"

	cred, err = cfg.getCredentials("the-other-registry")

	if err != nil {
		t.Errorf("Failed to extract credentials: %v", err)
	}

	if cred.User != expectedUser {
		t.Errorf("Expected user %s, actual was %s", expectedUser, cred.User)
	}

	if cred.Password != expectedPassword {
		t.Errorf("Expected password %s, actual was %s", expectedPassword, cred.Password)
	}

}
