package prepare

import (
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"github.com/skatteetaten/architect/v2/pkg/nexus"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

func TestArchitectPrepareLayers(t *testing.T) {
	buildconfiguration, err := prepareLayers(config.DockerSpec{
		OutputRegistry:         "",
		OutputRepository:       "",
		InternalPullRegistry:   "",
		PushExtraTags:          config.PushExtraTags{},
		ExternalDockerRegistry: "",
		TagWith:                "",
		RetagWith:              "",
	}, runtime.NewAuroraVersion("", false, "", ""),
		nexus.Deliverable{
			Path: "testdata/architect-1.25.14-Doozerleveransepakke.zip",
		},
		runtime.BaseImage{
			DockerImage: runtime.DockerImage{
				Tag:        "",
				Repository: "",
				Registry:   "",
			},
			ImageInfo: &runtime.ImageInfo{
				CompleteBaseImageVersion: "",
				Labels:                   map[string]string{},
				Enviroment:               nil,
				Digest:                   "",
			},
		})

	assert.NoError(t, err, "prepareLayers failed")

	path := buildconfiguration.BuildContext

	assert.FileExists(t, path+"/layer/u01/bin/architect")
	assert.Equal(t, "/u01/bin/architect", strings.Join(buildconfiguration.Cmd, " "))

	defer os.RemoveAll(path)

}

func TestPrepareLayers(t *testing.T) {
	buildconfiguration, err := prepareLayers(config.DockerSpec{
		OutputRegistry:         "",
		OutputRepository:       "",
		InternalPullRegistry:   "",
		PushExtraTags:          config.PushExtraTags{},
		ExternalDockerRegistry: "",
		TagWith:                "",
		RetagWith:              "",
	}, runtime.NewAuroraVersion("", false, "", ""),
		nexus.Deliverable{
			Path: "testdata/test-war-0.0.1-SNAPSHOT-DoozerLeveranse.zip",
		},
		runtime.BaseImage{
			DockerImage: runtime.DockerImage{
				Tag:        "",
				Repository: "",
				Registry:   "",
			},
			ImageInfo: &runtime.ImageInfo{
				CompleteBaseImageVersion: "",
				Labels:                   map[string]string{},
				Enviroment:               nil,
				Digest:                   "",
			},
		})

	assert.NoError(t, err, "prepareLayers failed")

	path := buildconfiguration.BuildContext

	t.Run("Check application structure", func(t *testing.T) {
		assert.DirExists(t, path+"/layer/u01")
		assert.FileExists(t, path+"/layer/usr/local/tomcat/webapps/emptywar.war")
	})

	os.RemoveAll(path)

}
