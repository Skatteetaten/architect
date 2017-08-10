package config_test

import (
	"github.com/docker/docker/pkg/testutil/assert"
	"github.com/skatteetaten/architect/pkg/config"
	"testing"
)

func TestJavaLeveransePakkeConfig(t *testing.T) {
	r := config.NewFileConfigReader("../../testdata/build.json")
	c, err := r.ReadConfig()
	if err != nil {
		t.Fatalf("Error when reading config: %s", err)
	}
	completeDockerName := c.DockerSpec.OutputRegistry + "/" + c.DockerSpec.OutputRepository
	assert.Equal(t, "docker-registry.themoon.com:5000/groupid/app", completeDockerName)
	assert.Equal(t, config.JavaLeveransepakke, c.ApplicationType)
	assert.Equal(t, "application-server", c.JavaApplication.ArtifactId)
	assert.Equal(t, "groupid.com", c.JavaApplication.GroupId)
	assert.Equal(t, "0.0.62", c.JavaApplication.Version)
}

func TestNodeJSLeveransePakkeConfig(t *testing.T) {
	r := config.NewFileConfigReader("../../testdata/nodejsbuild.json")
	c, err := r.ReadConfig()
	if err != nil {
		t.Fatalf("Error when reading config: %s", err)
	}
	completeDockerName := c.DockerSpec.OutputRegistry + "/" + c.DockerSpec.OutputRepository
	assert.Equal(t, "docker-registry.themoon.com:5000/groupid/app", completeDockerName)
	assert.Equal(t, config.NodeJsLeveransepakke, c.ApplicationType)
	assert.Equal(t, "nodejs-test-app", c.NodeJsApplication.NpmName)
	assert.Equal(t, "0.0.62", c.NodeJsApplication.Version)
}
