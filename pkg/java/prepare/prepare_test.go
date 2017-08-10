package prepare_test

import (
	global "github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/java/nexus"
	"github.com/skatteetaten/architect/pkg/java/prepare"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepare(t *testing.T) {
	auroraVersions := runtime.NewApplicationVersion(
		"2.0.0",
		true,
		"2.0.0",
		"2.0.0-b1.11.0-oracle8-1.0.2")

	dockerBuildPath, err := prepare.Prepare(global.DockerSpec{}, auroraVersions,
		&nexus.Deliverable{"testdata/minarch-1.2.22-Leveransepakke.zip"},
		&runtime.BaseImage{
			Repository: "test",
			Tag:        "1",
		})

	assert.NoError(t, err)

	// Test image scripts
	for _, script := range []string{"logback.xml", "run_tools.sh", "liveness_std.sh", "readiness_std.sh"} {
		scripPath := filepath.Join(dockerBuildPath, "app", "architect", script)
		scriptExists, err := prepare.Exists(scripPath)

		assert.NoError(t, err)
		assert.True(t, scriptExists, "Expected file "+scripPath+" not found")
	}

	// Dockerfile
	filePath := filepath.Join(dockerBuildPath, "Dockerfile")
	fileExists, err := prepare.Exists(filePath)

	if err != nil {
		t.Error(err)
	} else if !fileExists {
		t.Errorf("Expected file %s not found", filePath)
	}

	// Application
	applicationPath := filepath.Join(dockerBuildPath, "app", "application")
	applicationExists, err := prepare.Exists(applicationPath)

	if err != nil {
		t.Error(err)
	} else if !applicationExists {
		t.Errorf("Expected file %s not found", filePath)
	}

	os.RemoveAll(dockerBuildPath)

}
