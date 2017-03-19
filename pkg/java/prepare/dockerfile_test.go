package prepare

import (
	"testing"
	"bytes"
	"strings"
	"github.com/Skatteetaten/architect/pkg/java/config"
)

func TestBuild(t *testing.T) {

	const maintainer string = "tester@skatteetaten.no"
	const k8sDescription string = "Demo application with spring boot on Openshift."
	const openshiftTags string = "openshift,springboot"
	const readinessUrl string = "http://ready.skead.no"

	cfg := &config.ArchitectConfig{
		Docker: &struct {
			Maintainer string `json:"maintainer"`
			Labels     interface{} `json:"labels"`
		}{
			Maintainer: maintainer,
			Labels: map[string]interface{}{
				"io.k8s.description": k8sDescription,
				"io.openshift.tags":  openshiftTags,
			},
		},
		Openshift: &struct {
			ReadinessURL              string `json:"readinessUrl"`
			ReadinessOnManagementPort string `json:"readinessOnManagementPort"`
		}{
			ReadinessURL: readinessUrl,
		},
	}

	var buf bytes.Buffer

	NewForConfig("BaseUrl", "MY_ENV", cfg).Build(&buf)

	dockerfile := buf.String()

	assertContainsElement(t, dockerfile, maintainer)
	assertContainsElement(t, dockerfile, k8sDescription)
	assertContainsElement(t, dockerfile, openshiftTags)
	assertContainsElement(t, dockerfile, readinessUrl)
}

func assertContainsElement(t *testing.T, target string, element string) {
	if strings.Contains(target, element) == false {
		t.Error( "excpected", element, ", got", target)
	}
}
