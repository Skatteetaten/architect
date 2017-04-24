package prepare_test

import (
	"github.com/skatteetaten/architect/pkg/java/prepare"
	"testing"
	"path/filepath"
	"os"
	"github.com/skatteetaten/architect/pkg/docker"
	global "github.com/skatteetaten/architect/pkg/config"
)

func TestPrepare(t *testing.T) {

	ts, err := docker.StartMockRegistry()

	if err != nil {
		t.Fatal(err)
	}

	defer ts.Close()

	cfg := global.Config{"java",global.MavenGav{},
		global.DockerSpec{OutputImage:ts.URL, BaseImage:"aurora/oracle8:1"}}

	dockerBuildPath, err := prepare.Prepare(cfg, map[string]string{"VAR1": "VAL1", "VAR2": "VAL2"},
		"testdata/minarch-1.2.22-Leveransepakke.zip")

	if err != nil {
		t.Error(err)
	}

	// Test image scripts
	for _, script := range []string{"run", "run_tools.sh", "liveness_std.sh", "readiness_std.sh"} {
		scripPath := filepath.Join(dockerBuildPath, "app", "bin", script)
		scriptExists, err := prepare.Exists(scripPath)

		if err != nil {
			t.Error(err)
		} else if !scriptExists {
			t.Errorf("Expected file %s not found", scripPath)
		}
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

func TestFindRepoAndTagFromBaseImage(t *testing.T) {
	target := "a:b:c"

	s1, s2, err := prepare.FindRepoAndTagFromBaseImage(target, ":")

	if err != nil {
		t.Errorf("%v", err)
	}

	if s1 != "a:b" {
		t.Errorf("Expected first element %s, got %s", "a:b", s1 )
	}

	if s2 != "c" {
		t.Errorf("Expected last element %s, got %s", "c", s2 )
	}

}