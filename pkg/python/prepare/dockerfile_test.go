package prepare_test

import (
	"bytes"
	global "github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/python/config"
	"github.com/skatteetaten/architect/pkg/python/prepare"
	"github.com/stretchr/testify/assert"
	"testing"
)

const expectedDockerfile = `FROM rumple39:latest

MAINTAINER maintain@me.no
LABEL maintainer="maintain.me.no" no.skatteetaten.test="TestLabel" randomlabel="Use the 4ce"

# temp env hack until standard base image is available
ENV LANG='en_US.UTF-8' \
    TZ=Europe/Oslo \
    HOME=/u01

COPY ./app/application/bin $HOME/application/bin

RUN find $HOME/application -type d -exec chmod 755 {} + && \
	find $HOME/application -type f -exec chmod 644 {} + && \
	mkdir -p $HOME/logs && \
	chmod 777 $HOME/logs && \
	ln -s $HOME/logs $HOME/application/logs

ENV APP_VERSION="0.0.1-SNAPSHOT" AURORA_VERSION="0.0.1-SNAPSHOT-bbuildimage-rumple39-latest" IMAGE_BUILD_TIME="2017-09-10T14:30:10Z" PUSH_EXTRA_TAGS="major" SNAPSHOT_TAG="0.0.1-SNAPSHOT" TZ="Europe/Oslo"

CMD ["/u01/bin/run.sh"]
`

func TestBuild(t *testing.T) {
	dockerSpec := global.DockerSpec{
		PushExtraTags: global.ParseExtraTags("major"),
	}
	baseImage := runtime.DockerImage{
		Tag:        "latest",
		Repository: "rumple39",
	}
	auroraVersions := runtime.NewAuroraVersionFromBuilderAndBase(
		"0.0.1-SNAPSHOT",
		true,
		"0.0.1-SNAPSHOT",
		&runtime.ArchitectImage{
			Tag: "buildimage",
		},
		baseImage)

	labels := make(map[string]string)
	labels["no.skatteetaten.test"] = "TestLabel"
	labels["maintainer"] = "maintain.me.no"
	labels["randomlabel"] = "Use the 4ce"
	deliverableMetadata := config.DeliverableMetadata{
		Docker: &config.MetadataDocker{
			Maintainer: "maintain@me.no",
			Labels:     labels,
		},
		Python: &config.MetadataPython{},
	}
	writer := prepare.NewDockerFile(dockerSpec, *auroraVersions, deliverableMetadata, baseImage, "2017-09-10T14:30:10Z")

	buffer := new(bytes.Buffer)

	assert.NoError(t, writer(buffer))
	assert.Equal(t, expectedDockerfile, buffer.String())

}
