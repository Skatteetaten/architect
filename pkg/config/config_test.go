package config_test

import (
	"github.com/skatteetaten/architect/pkg/config"
	"testing"
)

func TestNewFileConfigReader(t *testing.T) {
	r := config.NewFileConfigReader("../../testdata/build.json")
	c, err := r.ReadConfig()
	if err != nil {
		t.Fatalf("Error when reading config: %s", err)
	}
	completeDockerName := c.DockerSpec.OutputRegistry + "/" + c.DockerSpec.OutputRepository
	if completeDockerName != "docker-registry.themoon.com:5000/groupid/app" {
		t.Errorf("Expected %s, was %s", "docker-registry.themoon.com:5000/groupid/app", completeDockerName)
	}
	if c.ApplicationType != config.JavaLeveransepakke {
		t.Errorf("Expected %s, was %s", config.JavaLeveransepakke, c.ApplicationType)
	}
	if c.MavenGav.ArtifactId != "application-server" {
		t.Errorf("Expected %s, was %s", "openshift-referanse-springboot-server", c.MavenGav.ArtifactId)
	}
	if c.MavenGav.GroupId != "groupid.com" {
		t.Errorf("Expected %s, was %s", "groupid.com", c.MavenGav.GroupId)
	}
	if c.MavenGav.Version != "0.0.62" {
		t.Errorf("Expected %s, was %s", "0.0.62", c.MavenGav.Version)
	}
}
