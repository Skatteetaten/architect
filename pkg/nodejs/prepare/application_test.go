package prepare_test

import (
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/nodejs/npm"
	"github.com/skatteetaten/architect/pkg/nodejs/prepare"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestApplicationPrepare(t *testing.T) {
	registryClient := npm.NewLocalRegistry("../npm/testfiles")

	c := config.Config{
		ApplicationType: config.NodeJsLeveransepakke,
		NodeJsApplication: &config.NodeApplication{
			NpmName: "openshift-referanse-react",
			Version: "0.1.2",
		},
	}

	imageInfoProvider := &testImageInfoProvider{}
	prepper := prepare.Prepper(registryClient)
	bc, err := prepper(&c, imageInfoProvider)
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
