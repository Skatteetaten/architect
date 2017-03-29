package prepare

import (
	"fmt"
	"github.com/skatteetaten/architect/pkg/java/config"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	DeliveryMetadataPath = "metadata/openshift.json"
)

func PrepareApplication(applicationPath string, meta *config.DeliverableMetadata) error {

	scriptPath := filepath.Join(applicationPath, "bin")

	if err := os.MkdirAll(scriptPath, 0755); err != nil {
		return err
	}

	libPath, err := findLibraryPath(applicationPath)

	if err != nil {
		return err
	}

	if meta != nil {
		if err := addGeneratedStartscript(scriptPath, libPath, *meta); err != nil {
			return err
		}
	}

	if err := prepareEffectiveStartscript(scriptPath); err != nil {
		return err
	}

	return nil
}

func addGeneratedStartscript(scriptPath string, libPath string, meta config.DeliverableMetadata) error {

	classpath, err := Classpath(libPath)

	if err != nil {
		return err
	}

	startscript := NewStartscript(classpath, meta)

	if err = WriteFile(filepath.Join(scriptPath, "generated-start"), startscript); err != nil {
		return err
	}

	return nil
}

func prepareEffectiveStartscript(scriptPath string) error {

	defaultScriptExists, err := Exists(filepath.Join(scriptPath, "os-start"))

	if err != nil {
		return err
	} else if defaultScriptExists {
		return nil
	}

	for _, altScriptName := range []string{"os-start.sh", "generated-start", "start", "start.sh"} {
		scriptExists, err := Exists(filepath.Join(scriptPath, altScriptName))

		if err != nil {
			return err
		} else if !scriptExists {
			continue
		}

		if err := os.Symlink(altScriptName, filepath.Join(scriptPath, "os-start")); err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("No start script found")
}

func Classpath(libPath string) ([]string, error) {

	files, err := ioutil.ReadDir(libPath)

	if err != nil {
		return nil, err
	}

	classpath := make([]string, len(files))

	for index, value := range files {
		classpath[index] = filepath.Join(libPath, value.Name())
	}

	return classpath, nil
}

func findLibraryPath(basedirPath string) (string, error) {

	for _, entry := range []string{"lib", "repo"} {
		libdirPath := filepath.Join(basedirPath, entry)

		exists, err := Exists(libdirPath)

		if err != nil {
			return "", err
		} else if exists {
			return libdirPath, nil
		}
	}

	return "", fmt.Errorf("No lib folder found")
}
