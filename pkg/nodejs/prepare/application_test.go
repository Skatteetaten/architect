package prepare_test

import (
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/nodejs/npm"
	"github.com/skatteetaten/architect/pkg/nodejs/prepare"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestApplicationPrepare(t *testing.T) {
	registryClient := npm.NewLocalRegistry("../npm/testfiles")

	c := config.Config{
		ApplicationType: config.NodeJsLeveransepakke,
		NodeJSGav: &config.NodeJSGav{
			NpmName: "openshift-referanse-react",
			Version: "0.1.2",
		},
	}
	dockerRegistry := docker.NewRegistryClient("test")

	prepper := prepare.Prepper(registryClient)
	bc, err := prepper(&c, dockerRegistry)
	for _, b := range bc {
		print(b.Tags)
		print(b.BuildFolder)
	}
	assert.NoError(t, err)
}
