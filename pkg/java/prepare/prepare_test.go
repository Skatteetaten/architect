package prepare_test

import (
	global "github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/java/config"
	"github.com/skatteetaten/architect/pkg/java/prepare"
	"os"
	"path/filepath"
	"testing"
)

var buildinfo = global.BuildInfo{
	Env: map[string]string{
		docker.ENV_APP_VERSION:     "2.0.0",
		docker.ENV_AURORA_VERSION:  "2.0.0-b1.11.0-oracle8-1.0.2",
		docker.ENV_PUSH_EXTRA_TAGS: "latest major minor patch"},
	OutputImage: global.OutputImageInfo{
		Repository: "foo/bar",
		TagInfo:    global.TagInfo{[]string{"2", "2.0", "2.0.0", "2.0.0-b1.11.0-oracle8-1.2.4", "latest"}}},
	BaseImage: global.BaseImageInfo{
		Repository: "aurora/oracle8",
		Version:    "1.2.4"},
}

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

	dockerBuildPath, err := prepare.Prepare(buildinfo,
		global.Deliverable{"testdata/minarch-1.2.22-Leveransepakke.zip"})

	if err != nil {
		t.Fatal(err)
	}

	// Test image scripts
	for _, script := range []string{"logback.xml", "run_tools.sh", "liveness_std.sh", "readiness_std.sh"} {
		scripPath := filepath.Join(dockerBuildPath, "app", "architect", script)
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
