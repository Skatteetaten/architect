package config_test

import (
	"github.com/docker/docker/pkg/testutil/assert"
	"github.com/skatteetaten/architect/pkg/config"
	"testing"
)

func TestJavaLeveransePakkeConfig(t *testing.T) {
	r := config.NewFileConfigReader("../../testdata/build.json")
	c, err := r.ReadConfig()
	assert.NilError(t, err)
	completeDockerName := c.DockerSpec.OutputRegistry + "/" + c.DockerSpec.OutputRepository
	assert.Equal(t, "docker-registry.themoon.com:5000/groupid/app", completeDockerName)
	assert.Equal(t, config.JavaLeveransepakke, c.ApplicationType)
	assert.Equal(t, "application-server", c.ApplicationSpec.MavenGav.ArtifactId)
	assert.Equal(t, "groupid.com", c.ApplicationSpec.MavenGav.GroupId)
	assert.Equal(t, "0.0.62", c.ApplicationSpec.MavenGav.Version)
}

func TestNodeJSLeveransePakkeConfig(t *testing.T) {
	r := config.NewFileConfigReader("../../testdata/nodejsbuild.json")
	c, err := r.ReadConfig()
	assert.NilError(t, err)
	completeDockerName := c.DockerSpec.OutputRegistry + "/" + c.DockerSpec.OutputRepository
	assert.Equal(t, "docker-registry.themoon.com:5000/groupid/app", completeDockerName)
	assert.Equal(t, config.NodeJsLeveransepakke, c.ApplicationType)
	assert.Equal(t, "nodejs-test-app", c.ApplicationSpec.MavenGav.ArtifactId)
	assert.Equal(t, "0.0.62", c.ApplicationSpec.MavenGav.Version)
	assert.Equal(t, "testgroup", c.ApplicationSpec.MavenGav.GroupId)
}

func TestTagWithConfig(t *testing.T) {
	r := config.NewFileConfigReader("../../testdata/build.json")
	c, err := r.ReadConfig()
	assert.NilError(t, err)
	completeDockerName := c.DockerSpec.OutputRegistry + "/" + c.DockerSpec.OutputRepository
	assert.Equal(t, "docker-registry.themoon.com:5000/groupid/app", completeDockerName)
	assert.Equal(t, "", c.DockerSpec.TagWith)
	r = config.NewFileConfigReader("../../testdata/build_tagwith.json")
	c, err = r.ReadConfig()
	assert.NilError(t, err)
	assert.Equal(t, "docker-registry.themoon.com:5000/groupid/app", completeDockerName)
	assert.Equal(t, "supertaggen", c.DockerSpec.TagWith)
}
