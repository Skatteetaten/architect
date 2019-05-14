package architect

import (
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/java/prepare"
	"github.com/skatteetaten/architect/pkg/nexus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestPrepareFiles(t *testing.T) {

	c := &config.Config{
		ApplicationType: "java",
		ApplicationSpec: config.ApplicationSpec{
			MavenGav: config.MavenGav{
				ArtifactId: "mf-folkesak",
				GroupId:    "ske.folkeregister.folkesak",
				Version:    "1.272.0",
				Classifier: "Leveransepakke",
				Type:       "zip",
			},
			BaseImageSpec: config.DockerBaseImageSpec{
				BaseImage:   "aurora/wingnut",
				BaseVersion: "latest",
			},
		},
		DockerSpec: config.DockerSpec{
			OutputRegistry:         "https://test",
			OutputRepository:       "test",
			ExternalDockerRegistry: "http://uil0paas-utv-registry01.skead.no:9090",
		},
	}

	dir, err := ioutil.TempDir("/tmp", "test")

	defer os.RemoveAll(dir)

	var nexusDownloader nexus.Downloader

	mavenRepo := "http://aurora/nexus/service/local/artifact/maven/content"

	nexusDownloader = nexus.NewPathAwareNexusDownloader(mavenRepo, dir)

	gav := c.ApplicationSpec.MavenGav

	deliverble, err := nexusDownloader.DownloadArtifact(&gav)

	//Generate docker files (radish for now)
	_, err = prepare.PrepareFiles(dir, deliverble, c)

	assert.NoError(t, err)

	files, _ := IOReadDir(dir)
	expected := []string{"Dockerfile", "app", "mf-folkesak-1.272.0-Leveransepakke.zip", "radish.json"}

	for i, got := range files {
		assert.Equal(t, expected[i], got)
	}
}

func IOReadDir(root string) ([]string, error) {
	var files []string
	fileInfo, err := ioutil.ReadDir(root)
	if err != nil {
		return files, err
	}

	for _, file := range fileInfo {
		files = append(files, file.Name())
	}
	return files, nil
}
