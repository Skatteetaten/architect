package prepare_test

import (
	"bytes"
	global "github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/java/config"
	"github.com/skatteetaten/architect/pkg/java/prepare"
	"github.com/stretchr/testify/assert"
	"testing"
)

const expectedDockerfile = `FROM test:oracle8-1.0.2

MAINTAINER wrench@sits.no
LABEL jallaball="Spank me beibi" maintainer="wrench.sits.no" no.skatteetaten.test="TestLabel"

COPY ./app $HOME
RUN chmod -R 777 $HOME && \
	ln -s $HOME/logs $HOME/application/logs && \
	rm $TRUST_STORE && \
	ln -s $HOME/architect/cacerts $TRUST_STORE

ENV APP_VERSION="2.0.0-SNAPSHOT" AURORA_VERSION="2.0.0-SNAPSHOT-b2.3.2-test-oracle8-1.0.2" LOGBACK_FILE="$HOME/architect/logback.xml" PUSH_EXTRA_TAGS=",,,," SNAPSHOT_TAG="2.0.0-SNAPSHOT" TZ="Europe/Oslo"
`

func TestBuild(t *testing.T) {
	dockerSpec := global.DockerSpec{
		BaseImage:   "test",
		BaseVersion: "tull",
	}
	auroraVersions, err := global.NewAuroraVersions(
		"2.0.0-SNAPSHOT",
		true,
		"2.0.0-SNAPSHOT",
		dockerSpec,
		global.BuilderSpec{
			Version: "2.3.2",
		},
		"oracle8-1.0.2")
	if err != nil {
		t.Fatal(err)
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
	writer := prepare.NewDockerfile(dockerSpec, auroraVersions, &deliverableMetadata)

	buffer := new(bytes.Buffer)

	assert.NoError(t, writer(buffer))
	assert.Equal(t, buffer.String(), expectedDockerfile)

}
