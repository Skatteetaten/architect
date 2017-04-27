package prepare_test

import (
	"github.com/skatteetaten/architect/pkg/java/prepare"
	"github.com/skatteetaten/architect/pkg/java/config"
	"testing"
	"path/filepath"
	"os"
	global "github.com/skatteetaten/architect/pkg/config"
)

var cfg global.Config = global.Config{"java", global.MavenGav{},
					   global.DockerSpec{BaseImage:"aurora/oracle8", BaseVersion: "1"},
					   global.BuilderSpec{}}

var buildinfo = global.BuildInfo{
	false,
	"1.2.1",
	global.ImageInfo{"aurora/beatie", "1.0.0", map[string]string{}},
	global.ImageInfo{"aurora/oracle8", "1.0.0",
			 map[string]string{"CONFIG_VERSION": "1", "INFERRED_VERSION": "1.2.3"}}}

var meta = &config.DeliverableMetadata{
	Docker: &struct {
		Maintainer string            `json:"maintainer"`
		Labels     map[string]string `json:"labels"`
	}{
		Maintainer: meta_maintainer,
		Labels: map[string]string{
			"io.k8s.description": meta_k8sDescription,
			"io.openshift.tags":  meta_openshiftTags,
		},
	},
	Openshift: &struct {
		ReadinessURL              string `json:"readinessUrl"`
		ReadinessOnManagementPort string `json:"readinessOnManagementPort"`
	}{
		ReadinessURL: meta_readinessUrl,
	},
}

var meta_maintainer string = "tester@skatteetaten.no"
var meta_k8sDescription string = "Demo application with spring boot on Openshift."
var meta_openshiftTags string = "openshift,springboot"
var meta_readinessUrl string = "http://ready.skead.no"

func TestPrepare(t *testing.T) {

	dockerBuildPath, err := prepare.Prepare(cfg, buildinfo,
		global.Deliverable{"testdata/minarch-1.2.22-Leveransepakke.zip"})

	if err != nil {
		t.Fatal(err)
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
