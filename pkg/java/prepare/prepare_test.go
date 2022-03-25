package prepare

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"github.com/skatteetaten/architect/v2/pkg/nexus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestPrepareLayers(t *testing.T) {

	buildConfiguration, err := prepareLayers(config.DockerSpec{
		OutputRegistry:         "https://localhost:5000",
		OutputRepository:       "aurora/minarch",
		InternalPullRegistry:   "https://localhost:5000",
		PushExtraTags:          config.PushExtraTags{},
		ExternalDockerRegistry: "https://localhost:5000",
		TagWith:                "latest",
		RetagWith:              "",
	}, runtime.NewAuroraVersion("", false, "", ""),
		nexus.Deliverable{
			Path: "testdata/minarch-1.2.22-Leveransepakke.zip",
			SHA1: "",
		},
	)

	if err != nil {
		t.Fatalf("Prepare layer failed with err=%s", err)
	}

	path := buildConfiguration.BuildContext

	t.Run("Check the application layer", func(t *testing.T) {

		assert.Nil(t, buildConfiguration.Cmd, "Should be empty")
		assert.DirExists(t, path+"/layer/u01", "layer does not exist")
		assert.DirExists(t, path+"/layer/u01/application", "The application folder does not exist")

		fileInfos, err := ioutil.ReadDir(path + "/layer/u01/application")
		if err != nil {
			t.Fatalf("Could not read application folder content")
		}

		var appContent []string
		for _, info := range fileInfos {
			appContent = append(appContent, info.Name())
		}

		assert.Equal(t, appContent, []string{"lib", "logs", "metadata"})
		assert.FileExists(t, path+"/layer/u01/application/metadata/openshift.json", "openshift.json is missing")

		fileInfos, err = ioutil.ReadDir(path + "/layer/u01/bin")
		if err != err {
			logrus.Fatal("Could not read the bin directory")
		}
		for _, info := range fileInfos {
			logrus.Infof("File: %s", info.Name())
		}

		fileInfos, err = ioutil.ReadDir(path + "/layer/u01/application/lib")
		if err != nil {
			t.Fatalf("Could not read application folder content")
		}

		var libContent []string
		for _, info := range fileInfos {
			libContent = append(libContent, info.Name())
		}

		assert.Equal(t, libContent, []string{"log4j-over-slf4j-1.7.6.jar", "minarch-1.2.22.jar", "slf4j-api-1.7.6.jar"}, "Wrong content in the classpath folder")

		fi, err := os.Lstat(path + "/layer/u01/application/logs")
		if err != nil {
			t.Fatal("Lstat failed")
		}
		isSymlink := fi.Mode()&os.ModeSymlink == os.ModeSymlink
		assert.NotEqual(t, isSymlink, "true", "Expected .../logs to be a symlink")

	})

	t.Run("Check radish.json", func(t *testing.T) {
		radishFile, _ := os.Open(path + "/layer/u01/radish.json")
		var javaDescriptor javaDescriptor
		err = json.NewDecoder(radishFile).Decode(&javaDescriptor)
		if err != nil {
			t.Fatal("Could not read the radish config")
		}
		assert.Equal(t, javaDescriptor.Type.Type, "Java", "Wrong type")
		assert.Equal(t, javaDescriptor.Type.Version, "1", "Wrong version")
		assert.Equal(t, javaDescriptor.Data.Basedir, "/u01/application", "Wrong basedir")
		assert.Equal(t, javaDescriptor.Data.ApplicationArgs, "--logging.config=${LOGBACK_FILE}", "Wrong application arguments")
		assert.Equal(t, javaDescriptor.Data.JavaOptions, "-Dspring.profiles.active=openshift", "Wrong java options")
		assert.Equal(t, strings.Join(javaDescriptor.Data.PathsToClassLibraries, ","), "lib,repo", "Wrong path to class libaries")

	})

	//Delete temp directory
	os.RemoveAll(path)

}
