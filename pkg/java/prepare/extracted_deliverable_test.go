package prepare

import (
	"os"
	"io"
	"testing"
	"io/ioutil"
	"path/filepath"
)

var orderedLibs []string = []string{"bar.jar", "bar2.jar", "foo.jar", "foobar.jar"}

var expectedStartscript string = "#!/bin/sh; echo start"

type StartScriptMock struct {}

func (startscript StartScriptMock) Write(writer io.Writer) error {
	_, err := writer.Write([]byte(expectedStartscript))
	return err
}

func TestClasspathOrder(t *testing.T) {

	// Given
	root, err := createTree()

	if err != nil {
		t.Error("Failed to create extracted deliverable for test: ", err)
	}

	target := NewExtractedDeliverable(root)

	// When
	actualCp, err := target.Classpath()

	// Then
	for idx, actualLib := range actualCp {
		expectedLib := filepath.Join(root, "lib", orderedLibs[idx])
		if expectedLib != actualLib {
			t.Error("excpected", expectedLib, ", got", actualLib)
		}
	}

	deleteTree(root)
}

func TestStartScript(t *testing.T) {

	// Given
	root, err := createTree()

	if err != nil {
		t.Error("Failed to create extracted deliverable for test: ", err)
	}

	target := NewExtractedDeliverable(root)

	// When
	target.AddStartscript(StartScriptMock{})

	// Then
	actualContent, err := ioutil.ReadFile(filepath.Join(root, "bin/generated-start"))

	if err != nil {
		t.Error("Failed to verify startscript: ", err)
	}

	if string(actualContent) != expectedStartscript {
		t.Error("excpected", expectedStartscript, ", got", string(actualContent))
	}

	//deleteTree(root)
}

func createTree() (string, error) {

	// Base folder
	root, err := ioutil.TempDir("", "architect-test")

	if err != nil {
		return "", err
	}

	// Sub folders
	for _, folder := range []string{"lib", "bin"} {
		err = os.MkdirAll(filepath.Join(root, folder), 0766)

		if err != nil {
			deleteTree(root)
			return "", err
		}
	}

	// Libraries
	for _, lib := range orderedLibs {
		err := ioutil.WriteFile(filepath.Join(root, "lib", lib), []byte("FOO"), 0666)

		if err != nil {
			deleteTree(root)
			return "", err
		}
	}

	return root, nil
}

func deleteTree(root string) error {
	return os.RemoveAll(root)
}
