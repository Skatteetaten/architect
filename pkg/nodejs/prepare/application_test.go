package prepare_test

import (
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/nexus"
	"github.com/skatteetaten/architect/pkg/nodejs/prepare"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestApplicationPrepare(t *testing.T) {

	c := config.Config{
		ApplicationType: config.NodeJsLeveransepakke,
		ApplicationSpec: config.ApplicationSpec{
			MavenGav: config.MavenGav{
				ArtifactId: "nodejs",
				GroupId:    "a.group.id",
				Version:    "0.1.2",
			},
		},
	}

	imageInfoProvider := &testImageInfoProvider{}
	auroraVersion := runtime.NewAuroraVersion("0.1.2", false, "0.1.2", runtime.CompleteVersion("0.1.2-b--baseimageversion"))
	prepper := prepare.Prepper()
	baseImage := runtime.BaseImage{
		DockerImage: runtime.DockerImage{
			Tag:        "test",
			Repository: "tull",
			Registry:   "tullogtoys",
		},
	}
	deliverable := nexus.Deliverable{Path: "testfiles/openshift-referanse-react-snapshot_test-SNAPSHOT-Webleveransepakke.tgz"}
	bc, err := prepper(&c, auroraVersion, deliverable, baseImage)
	assert.Equal(t, 1, len(bc))
	assert.NoError(t, err)
	b := bc[0]
	assert.Equal(t, "0.1.2-b--baseimageversion", b.AuroraVersion.GetCompleteVersion())
	assert.Equal(t, "0.1.2", string(b.AuroraVersion.GetAppVersion()))
	repositoryTags, _ := imageInfoProvider.GetTags("test")
	tags, err := b.AuroraVersion.GetApplicationVersionTagsToPush(repositoryTags.Tags, config.ParseExtraTags("latest major minor patch"))
	assert.NoError(t, err)
	assert.Equal(t, []string{"0.1", "0.1.2", "0.1.2-b--baseimageversion"}, tags)
	os.RemoveAll(b.BuildFolder)
}

type testImageInfoProvider struct {
}

func (m *testImageInfoProvider) GetCompleteBaseImageVersion(repository string, tag string) (string, error) {
	return "baseimageversion", nil
}
func (m *testImageInfoProvider) GetTags(repository string) (*docker.TagsAPIResponse, error) {
	return &docker.TagsAPIResponse{
		Name: repository,
		Tags: []string{"0", "0.2.0", "0.1.1"},
	}, nil
}
func (m *testImageInfoProvider) GetManifestEnvMap(repository string, tag string) (map[string]string, error) {
	return nil, nil
}
