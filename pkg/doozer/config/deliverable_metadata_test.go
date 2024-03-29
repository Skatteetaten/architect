package config

import (
	"strings"
	"testing"
)

func TestNewFromJson(t *testing.T) {

	const maintainer string = "Aurora OpenShift Utvikling <utvpaas@skatteetaten.no>"
	const readinessURL = "/health"
	const ioK8sDescription = "Demo application with spring boot on Openshift."
	const srcPath = "app/"
	const fileName = "application.war"
	const destPath = "/usr/local/tomcat/webapps/"

	const openshiftJSON string = `{
  "docker": {
    "maintainer": "Aurora OpenShift Utvikling <utvpaas@skatteetaten.no>",
    "labels": {
      "io.k8s.description": "Demo application with spring boot on Openshift.",
      "io.openshift.tags": "openshift,springboot"
    }
  },
  "doozer": {
    "srcPath": "app/",
    "fileName": "application.war",
    "destPath": "/usr/local/tomcat/webapps/",
    "entrypoint": "/bin/sh",
    "cmdScript": "-c ls -a"
  },
  "java": {
    "mainClass": "ske.aurora.openshift.referanse.springboot.Main",
    "jvmOpts": "-Dspring.profiles.active=openshift",
    "applicationArgs": "--logging.config=${LOGBACK_FILE}"
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
	assertEquals(t, srcPath, meta.Doozer.SrcPath)
	assertEquals(t, fileName, meta.Doozer.FileName)
	assertEquals(t, destPath, meta.Doozer.DestPath)
	assertEquals(t, "/bin/sh", meta.Doozer.Entrypoint)
	assertEquals(t, "-c ls -a", meta.Doozer.CmdScript)
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
		t.Error("expected", expected, ", got", actual)
	}
}
