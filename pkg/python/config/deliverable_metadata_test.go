package config

import (
	"strings"
	"testing"
)

func TestNewFromJson(t *testing.T) {

	const maintainer string = "Aurora OpenShift Utvikling <utvpaas@skatteetaten.no>"
	const readinessUrl = "/health"
	const ioK8sDescription = "Demo application with spring boot on Openshift."

	const openshiftJson string = `{
  "docker": {
    "maintainer": "Aurora OpenShift Utvikling <utvpaas@skatteetaten.no>",
    "labels": {
      "io.k8s.description": "Demo application with spring boot on Openshift.",
      "io.openshift.tags": "openshift,springboot"
    }
  },
  "python": {},
  "openshift" : {
      "readinessUrl": "/health"
  }
}`

	meta, err := NewDeliverableMetadata(strings.NewReader(openshiftJson))

	if err != nil {
		t.Error("Failed to initialize metadata from JSON")
	}

	assertEquals(t, maintainer, meta.Docker.Maintainer)
	assertEquals(t, readinessUrl, meta.Openshift.ReadinessURL)
	assertEquals(t, ioK8sDescription, meta.Docker.Labels["io.k8s.description"])
}

func TestErrorOnInvalidJson(t *testing.T) {
	const xml string = `<this>is not<json>`

	_, err := NewDeliverableMetadata(strings.NewReader(xml))

	if err == nil {
		t.Error("Invalid Json must return error")
	}
}

func assertEquals(t *testing.T, expected string, actual string) {
	if !(expected == actual) {
		t.Error("excpected", expected, ", got", actual)
	}
}
