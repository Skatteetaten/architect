package prepare_test

import (
	"bytes"
	"fmt"
	global "github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/java/config"
	"github.com/skatteetaten/architect/pkg/java/prepare"
	"github.com/stretchr/testify/assert"
	"testing"
)

const expectedDockerfile = `FROM oracle8:2.3.2

MAINTAINER wrench@sits.no
LABEL jallaball="Spank me beibi" maintainer="wrench.sits.no" no.skatteetaten.test="TestLabel"

COPY ./app $HOME
RUN chmod -R 777 $HOME && \
	ln -s $HOME/logs $HOME/application/logs && \
	rm $TRUST_STORE && \
	ln -s $HOME/architect/cacerts $TRUST_STORE

ENV FOO="BAR" TZ="2017-09-30T16:45:33Z"
`

const buildTime = "2017-08-28T11:13:30Z"

var expectedEnvMap = map[string]string{
	"LOGBACK_FILE":     "$HOME/architect/logback.xml",
	"APP_VERSION":      "2.0.0-SNAPSHOT",
	"AURORA_VERSION":   "2.0.0-SNAPSHOT-bbuildimage-oracle8-2.3.2",
	"PUSH_EXTRA_TAGS":  "major",
	"TZ":               "Europe/Oslo",
	"IMAGE_BUILD_TIME": "2017-08-28T11:13:30Z",
	"SNAPSHOT_TAG":     "2.0.0-SNAPSHOT",
}

func TestCreateEnv(t *testing.T) {
	baseImage := &runtime.DockerImage{
		Tag:        "2.3.2",
		Repository: "oracle8",
	}

	dockerSpec := global.DockerSpec{
		PushExtraTags: global.ParseExtraTags("major"),
	}

	auroraVersions := runtime.NewApplicationVersionFromBuilderAndBase(
		"2.0.0-SNAPSHOT",
		true,
		"2.0.0-SNAPSHOT",
		&runtime.ArchitectImage{
			Tag: "buildimage",
		},
		baseImage)

	labels := make(map[string]string)
	labels["no.skatteetaten.test"] = "TestLabel"
	labels["maintainer"] = "wrench.sits.no"
	labels["jallaball"] = "Spank me beibi"

	deliverableMetadata := config.DeliverableMetadata{
		Docker: &config.MetadataDocker{
			Maintainer: "wrench@sits.no",
			Labels:     labels,
		},
	}

	actualEnvMap := prepare.CreateEnv(auroraVersions, dockerSpec.PushExtraTags, &deliverableMetadata, buildTime)

	fmt.Println(actualEnvMap)

	verifyEnvMapContent(actualEnvMap, expectedEnvMap, t)
}

func TestBuild(t *testing.T) {

	baseImage := &runtime.DockerImage{
		Tag:        "2.3.2",
		Repository: "oracle8",
	}

	inputEnv := map[string]string{
		"FOO": "BAR",
		"TZ":  "2017-09-30T16:45:33Z",
	}

	labels := make(map[string]string)
	labels["no.skatteetaten.test"] = "TestLabel"
	labels["maintainer"] = "wrench.sits.no"
	labels["jallaball"] = "Spank me beibi"

	deliverableMetadata := config.DeliverableMetadata{
		Docker: &config.MetadataDocker{
			Maintainer: "wrench@sits.no",
			Labels:     labels,
		},
	}

	writer := prepare.NewDockerfile(&deliverableMetadata, baseImage, inputEnv)

	buffer := new(bytes.Buffer)

	assert.NoError(t, writer(buffer))
	assert.Equal(t, buffer.String(), expectedDockerfile)

}

func verifyEnvMapContent(actualMap map[string]string, expectedMap map[string]string, t *testing.T) {
	for k, e := range expectedMap {
		verifyEnvMapContains(actualMap, k, e, t)
	}
}

func verifyEnvMapContains(actualMap map[string]string, key string, expected string, t *testing.T) {
	actual, ok := actualMap[key]

	if !ok {
		t.Errorf("Env map does not contain variable %s", key)
		return
	}

	if actual != expected {
		t.Errorf("Expected env value %s, actual is %s", expected, actual)
	}
}
