package prepare_test

import (
	"bytes"
	global "github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/doozer/config"
	"github.com/skatteetaten/architect/pkg/doozer/prepare"
	"github.com/stretchr/testify/assert"
	"testing"
)

const expectedDockerfile = `FROM tomcat:8.5.47-jdk11-openjdk

MAINTAINER maintain@me.no
LABEL maintainer="maintain.me.no" no.skatteetaten.test="TestLabel" randomlabel="Use the 4ce"

# temp env hack until standard base image is available
ENV LANG='en_US.UTF-8' \
    TZ=Europe/Oslo \
    HOME=/u01

COPY ./app radish.json $HOME/
COPY ./app/application/app/uberfile.war /path/to/your/destiny/uberfile.war

RUN find $HOME/application -type d -exec chmod 755 {} + && \
	find $HOME/application -type f -exec chmod 644 {} + && \
	mkdir -p $HOME/logs && \
	chmod 777 $HOME/logs && \
	ln -s $HOME/logs $HOME/application/logs

ENV APP_VERSION="0.0.1-SNAPSHOT" AURORA_VERSION="0.0.1-SNAPSHOT-bbuildimage-tomcat-8.5.47-jdk11-openjdk" IMAGE_BUILD_TIME="2017-09-10T14:30:10Z" PUSH_EXTRA_TAGS="major" SNAPSHOT_TAG="0.0.1-SNAPSHOT" TZ="Europe/Oslo"
`

const expectedDockerfileWithCmdScript = `FROM builder:latest

MAINTAINER maintain@me.no
LABEL maintainer="maintain.me.no" no.skatteetaten.test="TestLabel" randomlabel="Use the 4ce"

# temp env hack until standard base image is available
ENV LANG='en_US.UTF-8' \
    TZ=Europe/Oslo \
    HOME=/u01

COPY ./app radish.json $HOME/
COPY ./app/application/app/uberfile.war /path/to/your/destiny/uberfile.war

RUN find $HOME/application -type d -exec chmod 755 {} + && \
	find $HOME/application -type f -exec chmod 644 {} + && \
	mkdir -p $HOME/logs && \
	chmod 777 $HOME/logs && \
	ln -s $HOME/logs $HOME/application/logs

ENV APP_VERSION="0.0.1-SNAPSHOT" AURORA_VERSION="0.0.1-SNAPSHOT-bbuildimage-builder-latest" IMAGE_BUILD_TIME="2017-09-10T14:30:10Z" PUSH_EXTRA_TAGS="major" SNAPSHOT_TAG="0.0.1-SNAPSHOT" TZ="Europe/Oslo"
CMD ["./bin/somestartupcmd"]
`

func TestBuild(t *testing.T) {
	dockerSpec := global.DockerSpec{
		PushExtraTags: global.ParseExtraTags("major"),
	}
	baseImage := runtime.DockerImage{
		Tag:        "8.5.47-jdk11-openjdk",
		Repository: "tomcat",
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
		Doozer: &config.MetadataDoozer{
			SrcPath:  "app/",
			FileName: "uberfile.war",
			DestPath: "/path/to/your/destiny/",
		},
	}
	writer := prepare.NewDockerFile(dockerSpec, *auroraVersions, deliverableMetadata, baseImage, "2017-09-10T14:30:10Z", "")

	buffer := new(bytes.Buffer)

	assert.NoError(t, writer(buffer))
	assert.Equal(t, expectedDockerfile, buffer.String())

}

func TestBuildWithCmdScript(t *testing.T) {
	dockerSpec := global.DockerSpec{
		PushExtraTags: global.ParseExtraTags("major"),
	}
	baseImage := runtime.DockerImage{
		Tag:        "latest",
		Repository: "builder",
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
		Doozer: &config.MetadataDoozer{
			SrcPath:   "app/",
			FileName:  "uberfile.war",
			DestPath:  "/path/to/your/destiny/",
			CmdScript: "./bin/somestartupcmd",
		},
	}
	writer := prepare.NewDockerFile(dockerSpec, *auroraVersions, deliverableMetadata, baseImage, "2017-09-10T14:30:10Z", "")

	buffer := new(bytes.Buffer)

	assert.NoError(t, writer(buffer))
	assert.Equal(t, expectedDockerfileWithCmdScript, buffer.String())

}
