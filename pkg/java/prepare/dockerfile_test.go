package prepare

import (
	"testing"
	"bytes"
	"github.com/Skatteetaten/architect/pkg/java/config"
	"github.com/docker/docker/pkg/testutil/assert"
)

func TestBuild(t *testing.T) {

	const maintainer  string = "tester@skatteetaten.no"
	const k8sDescription string = "Demo application with spring boot on Openshift."
	const openshiftTags string = "openshift,springboot"
	const readinessUrl string = "http://ready.skead.no"

	cfg := &config.ArchitectConfig{
		Docker: struct {
			Maintainer string `json:"maintainer"`
			Labels interface{} `json:"labels"`
		}{
			Maintainer: maintainer,
			Labels: map[string]interface{}{
				"io.k8s.description": k8sDescription,
				"io.openshift.tags": openshiftTags,
			},
		},
		Openshift: struct {
			ReadinessURL string `json:"readinessUrl"`
			ReadinessOnManagementPort string `json:"readinessOnManagementPort"`
		}{
			ReadinessURL: readinessUrl,
		},
	}

	var buf bytes.Buffer

	NewForConfig(cfg).Build("Base", "BAYA", &buf)

	assert.Contains(t, buf.String(), maintainer)
	assert.Contains(t, buf.String(), k8sDescription)
	assert.Contains(t, buf.String(), openshiftTags)
	assert.Contains(t, buf.String(), readinessUrl)
}
