package config_test

import (
	"testing"
	"github.com/skatteetaten/architect/pkg/config"
)

func TestNewFileConfigReader(t *testing.T) {
	r := config.NewFileConfigReader("../../testdata/build.json")
	c, err := r.ReadConfig()
	if err != nil {
		t.Fatalf("Error when reading config: %s", err)
	}
	if c.DockerSpec.Registry != "docker-registry.themoon.com:5000/groupid/app" {
		t.Errorf("Expected %s, was %s", "docker-registry.themoon.com:5000/groupid/app", c.DockerSpec.Registry)
	}
	if c.ApplicationType != config.JavaLeveransepakke {
		t.Errorf("Expected %s, was %s", config.JavaLeveransepakke, c.ApplicationType)
	}
	if c.NexusGav.ArtifactId != "application-server" {
		t.Errorf("Expected %s, was %s", "openshift-referanse-springboot-server", c.NexusGav.ArtifactId)
	}
	if c.NexusGav.GroupId != "groupid.com" {
		t.Errorf("Expected %s, was %s", "groupid.com", c.NexusGav.GroupId)
	}
	if c.NexusGav.Version != "0.0.62" {
		t.Errorf("Expected %s, was %s", "0.0.62", c.NexusGav.Version)
	}
}
