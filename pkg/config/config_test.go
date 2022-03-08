package config_test

import (
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJavaLeveransePakkeConfig(t *testing.T) {
	r := config.NewFileConfigReader("../../testdata/build.json")
	c, err := r.ReadConfig()
	assert.NoError(t, err)
	completeDockerName := c.DockerSpec.OutputRegistry + "/" + c.DockerSpec.OutputRepository
	assert.Equal(t, "docker-registry.themoon.com:5000/groupid/app", completeDockerName)
	assert.Equal(t, config.JavaLeveransepakke, c.ApplicationType)
	assert.Equal(t, "application-server", c.ApplicationSpec.MavenGav.ArtifactId)
	assert.Equal(t, "groupid.com", c.ApplicationSpec.MavenGav.GroupId)
	assert.Equal(t, "0.0.62", c.ApplicationSpec.MavenGav.Version)
	assert.Equal(t, "1", c.ApplicationSpec.BaseImageSpec.BaseVersion)
}

func TestNodeJSLeveransePakkeConfig(t *testing.T) {
	r := config.NewFileConfigReader("../../testdata/nodejsbuild.json")
	c, err := r.ReadConfig()

	assert.NoError(t, err)
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
	assert.NoError(t, err)
	completeDockerName := c.DockerSpec.OutputRegistry + "/" + c.DockerSpec.OutputRepository
	assert.Equal(t, "docker-registry.themoon.com:5000/groupid/app", completeDockerName)
	assert.Equal(t, "", c.DockerSpec.TagWith)
	r = config.NewFileConfigReader("../../testdata/build_tagwith.json")
	c, err = r.ReadConfig()
	assert.NoError(t, err)
	assert.Equal(t, "docker-registry.themoon.com:5000/groupid/app", completeDockerName)
	assert.Equal(t, "supertaggen", c.DockerSpec.TagWith)
}

func TestHidingPasswordWhenGettingNExusAccessString(t *testing.T) {
	nexusAccess := config.NexusAccess{}
	nexusAccess.Username = "username"
	nexusAccess.Password = "password1"
	nexusAccess.NexusURL = "http://testurl"

	assert.Contains(t, nexusAccess.String(), "username")
	assert.Contains(t, nexusAccess.String(), "http://testurl")
	assert.Contains(t, nexusAccess.String(), "******")
}
func TestGetUrlFromOutput(t *testing.T) {
	r := config.NewFileConfigReader("../../testdata/bug-sitj-650.json")
	c, err := r.ReadConfig()
	assert.NoError(t, err)
	completeDockerName := c.DockerSpec.OutputRegistry + "/" + c.DockerSpec.OutputRepository
	assert.Equal(t, "container-registry-internal-snapshot.aurora.skead.no:443/no_skatteetaten_aurora_openshift/openshift-reference-springboot-server-kotlin", completeDockerName)
}
