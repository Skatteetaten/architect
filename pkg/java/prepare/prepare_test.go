package prepare_test

import (
	global "github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/java/prepare"
	"github.com/skatteetaten/architect/pkg/nexus"
	"github.com/skatteetaten/architect/pkg/util"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepare(t *testing.T) {
	auroraVersions := runtime.NewAuroraVersion(
		"2.0.0",
		true,
		"2.0.0",
		"2.0.0-b1.11.0-oracle8-1.0.2")

	dockerBuildPath, err := prepare.Prepare(global.DockerSpec{}, auroraVersions,
		nexus.Deliverable{Path: "testdata/minarch-1.2.22-Leveransepakke.zip"},
		runtime.BaseImage{
			DockerImage: runtime.DockerImage{
				Repository: "test",
				Tag:        "1",
			},
			ImageInfo: &runtime.ImageInfo{
				CompleteBaseImageVersion: "hei",
				Enviroment:               make(map[string]string),
				Labels:                   map[string]string{"www.skatteetaten.no-imageArchitecture": "java"},
			},
		})

	assert.NoError(t, err)

	// Dockerfile
	filePath := filepath.Join(dockerBuildPath, "Dockerfile")
	fileExists, err := util.Exists(filePath)

	//radish
	filePath = filepath.Join(dockerBuildPath, "radish.json")
	fileExists, err = util.Exists(filePath)

	if err != nil {
		t.Error(err)
	} else if !fileExists {
		t.Errorf("Expected file %s not found", filePath)
	}

	// Application
	applicationPath := filepath.Join(dockerBuildPath, "app", "application")
	applicationExists, err := util.Exists(applicationPath)

	if err != nil {
		t.Error(err)
	} else if !applicationExists {
		t.Errorf("Expected file %s not found", filePath)
	}

	os.RemoveAll(dockerBuildPath)

}
