package trace_test

import (
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"github.com/skatteetaten/architect/v2/pkg/java/prepare"
	"github.com/skatteetaten/architect/v2/pkg/nexus"
	"github.com/skatteetaten/architect/v2/pkg/trace"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScanImage(t *testing.T) {
	prepper := prepare.Prepper()
	buildConfig, err := prepper(&config.Config{
		ApplicationType: "",
		ApplicationSpec: config.ApplicationSpec{},
		DockerSpec: config.DockerSpec{
			OutputRegistry:         "https://localhost:5000",
			OutputRepository:       "aurora/minarch",
			InternalPullRegistry:   "https://localhost:5000",
			PushExtraTags:          config.PushExtraTags{},
			ExternalDockerRegistry: "https://localhost:5000",
			TagWith:                "latest",
			RetagWith:              "",
		},
		BuilderSpec:        config.BuilderSpec{},
		BinaryBuild:        false,
		LocalBuild:         false,
		TLSVerify:          false,
		BuildTimeout:       0,
		NoPush:             false,
		Sporingstjeneste:   "",
		OwnerReferenceUUID: "",
		BinaryBuildType:    "",
		NexusIQReportURL:   "",
	},
		runtime.NewAuroraVersion("", false, "", ""),
		nexus.Deliverable{
			Path: "testdata/minarch-1.2.22-Leveransepakke.zip",
			SHA1: "",
		},
		runtime.BaseImage{DockerImage: runtime.DockerImage{}},
	)
	assert.NoError(t, err)
	var traceClient = trace.NewClient("url")
	dependencies, err := traceClient.ScanImage(buildConfig.BuildFolder)
	assert.NoError(t, err)
	var dep1 = trace.Dependency{Purl: "pkg:maven/org.slf4j/slf4j-api@1.7.6",
		DependencyId:      "93824bab1fb3d6e0",
		Name:              "slf4j-api",
		Version:           "1.7.6",
		ChecksumAlgorithm: "sha1",
		ChecksumValue:     "562424e36df3d2327e8e9301a76027fca17d54ea"}

	assert.Equal(t, dependencies[2].Purl, dep1.Purl)
	assert.Equal(t, dependencies[2].DependencyId, dep1.DependencyId)
	assert.Equal(t, dependencies[2].Name, dep1.Name)
	assert.Equal(t, dependencies[2].Version, dep1.Version)
	assert.Equal(t, dependencies[2].ChecksumAlgorithm, dep1.ChecksumAlgorithm)
	assert.Equal(t, dependencies[2].ChecksumValue, dep1.ChecksumValue)
	assert.Contains(t, dependencies[2].SourceLocation, "u01/application/lib/slf4j-api-1.7.6.jar")
}
