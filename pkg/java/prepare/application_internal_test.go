package prepare

import (
	"github.com/docker/docker/pkg/ioutils"
	"github.com/skatteetaten/architect/pkg/java/config"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

func TestClasspathOrder(t *testing.T) {

	// Given
	root := setupApplication(t)

	// When
	actualCp, err := generateClasspath(root, filepath.Join(root, "lib"))

	if err != nil {
		t.Fatal("Failed to test classpath: ", err)
	}

	if len(actualCp) < 1 {
		t.Fatal("Classpath has zero entries")
	}

	// Then
	for idx, entry := range []string{"bar.jar", "bar2.jar", "foo.jar", "foobar.jar"} {
		expectedLib := filepath.Join("$HOME", "application", "lib", entry)
		if idx >= len(actualCp) {
			t.Error("Classpath", actualCp, "is not complete")
			return
		}

		actualLib := actualCp[idx]
		if actualLib != expectedLib {
			t.Error("Classpath", actualCp, "is not correct, excpected", expectedLib, ", got", actualLib)
		}
	}

	for _, entry := range actualCp {
		if strings.HasPrefix(entry, root) {
			t.Fatal("Class path-entries should not have root prefix")
		}
	}

	deleteApplication(root)
}

func TestPrepareStartscript(t *testing.T) {

	// Given
	root := setupApplication(t)

	meta := &config.DeliverableMetadata{
		Java: &config.MetadataJava{
			MainClass: "test",
		},
	}
	// When
	err := prepareEffectiveScripts(root, meta)

	assert.NoError(t, err)
	// Then
	scriptExists, err := Exists(filepath.Join(root, "bin", "generated-start"))

	assert.NoError(t, err)
	assert.True(t, scriptExists, "Failed to generate startscript")

	linkExists, err := Exists(filepath.Join(root, "bin", "start"))

	assert.NoError(t, err)
	assert.True(t, linkExists, "Failed to generate link to startscript")

	deleteApplication(root)
}

func TestPrepareWithoutGeneratedStartScript(t *testing.T) {

	var emptyFile = make([]byte, 0, 0)

	scriptsInPrecedenceOrder := []string{"os-start.sh", "os-start", "start.sh"}

	for i, startScript := range scriptsInPrecedenceOrder {
		root := setupApplication(t)
		for _, lowerAndEqualPrecedence := range scriptsInPrecedenceOrder[i:] {
			err := ioutils.AtomicWriteFile(path.Join(root, "bin", lowerAndEqualPrecedence), emptyFile, 0755)
			assert.NoError(t, err, "Unable to create script file")
		}
		err := prepareEffectiveScripts(root, &config.DeliverableMetadata{})
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(root, "bin", "generated-start"))
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
		fileInfo, err := os.Readlink(filepath.Join(root, "bin", "start"))
		assert.NoError(t, err)
		assert.Equal(t, startScript, fileInfo, "Failed to generate link to startscript")
		deleteApplication(root)
	}
}

func setupApplication(t *testing.T) string {

	// Base folder
	root, err := ioutil.TempDir("", "architect-test")

	if err != nil {
		t.Fatal("Failed to create test application: ", err)
	}

	// Sub folders
	for _, folder := range []string{"lib", "bin"} {
		err = os.MkdirAll(filepath.Join(root, folder), 0766)

		if err != nil {
			deleteApplication(root)
			t.Fatal("Failed to create test application: ", err)
		}
	}

	// Libraries
	for _, lib := range []string{"foo.jar", "bar.jar", "foobar.jar", "bar2.jar", "foo.jar"} {
		err := ioutil.WriteFile(filepath.Join(root, "lib", lib), []byte("FOO"), 0666)

		if err != nil {
			deleteApplication(root)
			t.Fatal("Failed to create test application: ", err)
		}
	}

	return root
}

func deleteApplication(root string) error {
	return os.RemoveAll(root)
}
