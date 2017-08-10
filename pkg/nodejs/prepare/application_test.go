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

	prepper := prepare.Prepper(registryClient)
	bc, err := prepper(&c, &testImageInfoProvider{})
	assert.Equal(t, 2, len(bc))
	for _, b := range bc {
		os.RemoveAll(b.BuildFolder)
	}
	assert.NoError(t, err)
}

type testImageInfoProvider struct {
}

func (m *testImageInfoProvider) GetCompleteBaseImageVersion(repository string, tag string) (string, error) {
	return "22", nil
}
func (m *testImageInfoProvider) GetTags(repository string) (*docker.TagsAPIResponse, error) {
	return nil, nil
}
func (m *testImageInfoProvider) GetManifestEnvMap(repository string, tag string) (map[string]string, error) {
	return nil, nil
}
