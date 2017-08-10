package prepare_test

import (
	"bytes"
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

ENV APP_VERSION="2.0.0-SNAPSHOT" AURORA_VERSION="2.0.0-SNAPSHOT-bbuildimage-oracle8-2.3.2" LOGBACK_FILE="$HOME/architect/logback.xml" PUSH_EXTRA_TAGS="major" SNAPSHOT_TAG="2.0.0-SNAPSHOT" TZ="Europe/Oslo"
`

func TestBuild(t *testing.T) {
	dockerSpec := global.DockerSpec{
		PushExtraTags: global.ParseExtraTags("major"),
	}
	baseImage := &runtime.BaseImage{
		Tag:        "2.3.2",
		Repository: "oracle8",
	}
	auroraVersions := runtime.NewApplicationVersionFromBuilderAndBase(
		"2.0.0-SNAPSHOT",
		true,
		"2.0.0-SNAPSHOT",
		&runtime.BuildImage{
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
	writer := prepare.NewDockerfile(dockerSpec, auroraVersions, &deliverableMetadata, baseImage)

	buffer := new(bytes.Buffer)

	assert.NoError(t, writer(buffer))
	assert.Equal(t, buffer.String(), expectedDockerfile)

}
