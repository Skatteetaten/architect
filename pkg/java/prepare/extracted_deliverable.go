package prepare

import (
	"bufio"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

type ExtractedDeliverable struct {
	baseFolder string
}

func NewExtractedDeliverable(baseFolder string) ExtractedDeliverable {
	return ExtractedDeliverable{baseFolder}
}

func (artifact *ExtractedDeliverable) Classpath() ([]string, error) {

	libFolder, err := artifact.locateLibraryPath()

	if err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(libFolder)

	if err != nil {
		return nil, err
	}

	classpath := make([]string, len(files))

	for index, value := range files {
		classpath[index] = filepath.Join(libFolder, value.Name())
	}

	return classpath, err
}

func (artifact *ExtractedDeliverable) AddStartscript(script StartScript) error {

	path := filepath.Join(artifact.baseFolder, "bin/generated-start")

	scriptFile, err := os.Create(path)

	if err != nil {
		return err
	}

	defer scriptFile.Close()

	writer := bufio.NewWriter(scriptFile)

	err = script.Write(writer)

	if err != nil {
		return err
	}

	return writer.Flush()
}

func (artifact *ExtractedDeliverable) locateLibraryPath() (folder string, err error) {
	if _, err := os.Stat(filepath.Join(artifact.baseFolder, "repo")); err == nil {
		folder = filepath.Join(artifact.baseFolder, "repo")
	} else if _, err := os.Stat(filepath.Join(artifact.baseFolder, "lib")); err == nil {
		folder = filepath.Join(artifact.baseFolder, "lib")
	} else {
		err = errors.New("No lib folder in artifact")
	}

	return folder, err
}
