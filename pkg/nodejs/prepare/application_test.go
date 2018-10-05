package prepare_test

import (
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
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
	os.RemoveAll(b.BuildFolder)
}
