package prepare

import (
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/nexus"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestPrepareLayers(t *testing.T) {

	buildConfiguration, err := prepareLayers(config.DockerSpec{
		OutputRegistry:         "",
		OutputRepository:       "",
		InternalPullRegistry:   "",
		PushExtraTags:          config.PushExtraTags{},
		ExternalDockerRegistry: "",
		TagWith:                "",
		RetagWith:              "",
		TagOverwrite:           false,
	}, runtime.NewAuroraVersion("", false, "", ""),
		nexus.Deliverable{
			Path: "testdata/openshift-referanse-react-snapshot_test-SNAPSHOT-Webleveransepakke.tgz",
			SHA1: "",
		}, runtime.BaseImage{
			DockerImage: runtime.DockerImage{},
			ImageInfo:   nil,
		})

	assert.NoError(t, err, "Prepare layers failed")

	path := buildConfiguration.BuildContext

	t.Run("Check application layer", func(t *testing.T) {
		assert.DirExists(t, path+"/layer/u01")

		assert.DirExists(t, path+"/layer/u01/application")
		assert.DirExists(t, path+"/layer/u01/application/api")
		assert.DirExists(t, path+"/layer/u01/application/build")
		assert.DirExists(t, path+"/layer/u01/application/metadata")
		assert.DirExists(t, path+"/layer/u01/application/node_modules")
		assert.FileExists(t, path+"/layer/u01/application/package.json")
		assert.FileExists(t, path+"/layer/u01/application/README.md")

		assert.DirExists(t, path+"/layer/u01/bin")
		assert.FileExists(t, path+"/layer/u01/bin/liveness_nginx.sh")
		assert.FileExists(t, path+"/layer/u01/bin/readiness_nginx.sh")
		assert.FileExists(t, path+"/layer/u01/bin/liveness_node.sh")
		assert.FileExists(t, path+"/layer/u01/bin/readiness_node.sh")

		assert.DirExists(t, path+"/layer/u01/logs")

		assert.DirExists(t, path+"/layer/u01/static/static")
		assert.DirExists(t, path+"/layer/u01/static/assets")
		assert.FileExists(t, path+"/layer/u01/static/asset-manifest.json")
		assert.FileExists(t, path+"/layer/u01/static/favicon.ico")
		assert.FileExists(t, path+"/layer/u01/static/manifest.json")
		assert.FileExists(t, path+"/layer/u01/static/index.html")
		assert.FileExists(t, path+"/layer/u01/static/service-worker.js")

	})

	os.RemoveAll(path)

}
