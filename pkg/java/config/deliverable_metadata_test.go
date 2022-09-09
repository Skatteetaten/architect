package config

import (
	"strings"
	"testing"
)

func TestNewFromJson(t *testing.T) {

	const maintainer string = "Aurora OpenShift Utvikling <utvpaas@skatteetaten.no>"
	const readinessURL = "/health"
	const ioK8sDescription = "Demo application with spring boot on Openshift."
	const startScript = "start.sh"
	const javaOpts = "-Dspring.profiles.active=openshift"
	const applicationArgs = "--logging.config=${LOGBACK_FILE}"
	const mainClass = "ske.aurora.openshift.referanse.springboot.Main"

	const openshiftJSON string = `{
  "docker": {
    "maintainer": "Aurora OpenShift Utvikling <utvpaas@skatteetaten.no>",
    "labels": {
      "io.k8s.description": "Demo application with spring boot on Openshift.",
      "io.openshift.tags": "openshift,springboot"
    }
  },
  "java": {
    "mainClass": "ske.aurora.openshift.referanse.springboot.Main",
    "jvmOpts": "-Dspring.profiles.active=openshift",
    "applicationArgs": "--logging.config=${LOGBACK_FILE}", 
    "startScript": "start.sh"
  },
  "openshift" : {
      "readinessUrl": "/health"
  }
}`

	meta, err := NewDeliverableMetadata(strings.NewReader(openshiftJSON))

	if err != nil {
		t.Error("Failed to initialize metadata from JSON")
	}

	assertEquals(t, maintainer, meta.Docker.Maintainer)
	assertEquals(t, readinessURL, meta.Openshift.ReadinessURL)
	assertEquals(t, ioK8sDescription, meta.Docker.Labels["io.k8s.description"])
	assertEquals(t, startScript, meta.Java.StartScript)
	assertEquals(t, javaOpts, meta.Java.JvmOpts)
	assertEquals(t, applicationArgs, meta.Java.ApplicationArgs)
	assertEquals(t, mainClass, meta.Java.MainClass)
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
