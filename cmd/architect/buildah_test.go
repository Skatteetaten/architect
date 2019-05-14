package architect

import (
	"github.com/Sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/java"
	"github.com/skatteetaten/architect/pkg/nexus"
	process "github.com/skatteetaten/architect/pkg/process/build"
	"testing"
)

type DummyImageInfoProvider struct {
}

type DummyDownloader struct {
}

type DoNothingBuilder struct {

}

func (dummy DummyImageInfoProvider) GetImageInfo(repository string, tag string) (*runtime.ImageInfo, error) {
	return &runtime.ImageInfo{
		"",
		nil,
		nil,
	}, nil
}

func (dummy DummyImageInfoProvider) GetTags(repository string) (*docker.TagsAPIResponse, error) {
	return &docker.TagsAPIResponse{
		"jalla",
		[]string{"latest"},
	}, nil
}

func (dummy DummyDownloader) DownloadArtifact(c *config.MavenGav) (nexus.Deliverable, error) {
	return nexus.Deliverable{
		"testdata/minarch-1.2.22-Leveransepakke.zip",
	}, nil
}

func (dummy DoNothingBuilder) Build(ruuid, context, buildFolder string) error{
	logrus.Infof("Calling buildah with buildcontext %s and buildfolder %s",context, buildFolder )
	return nil;
}

func (dummy DoNothingBuilder) Push(ruuid, tag string) error {
	logrus.Infof("pushing image with tag %s", tag)
	return nil
}

func (dummy DoNothingBuilder) Tag(ruuid, tag string) error {
	logrus.Infof("Tagging image %s", tag)
	return nil
}



func TestBuildWithBuildah(t *testing.T) {

	provider := DummyImageInfoProvider{}
	downloader := DummyDownloader{}

	c := &config.Config{
		ApplicationType:"JavaLeveransepakke",

	}
	prepper := java.Prepper()
	builder := DoNothingBuilder{}

	err := process.Build(nil, provider, c, downloader, prepper, builder)

	if err != nil {
		t.Errorf("Buildah operation failed: %s", err)
	}

}
