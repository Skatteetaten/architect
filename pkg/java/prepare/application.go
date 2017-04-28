package prepare

import (
	"fmt"
	"github.com/skatteetaten/architect/pkg/java/config"
	"io/ioutil"
	"os"
	"path/filepath"
	"github.com/pkg/errors"
)

const (
	DeliveryMetadataPath = "metadata/openshift.json"
)

func PrepareApplication(applicationPath string, meta *config.DeliverableMetadata) error {

	scriptPath := filepath.Join(applicationPath, "bin")

	if err := os.MkdirAll(scriptPath, 0755); err != nil {
		return errors.Wrap(err, "Failed to create directory")
	}

	libPath, err := findLibraryPath(applicationPath)

	if err != nil {
		return errors.Wrap(err, "Failed to locate lib directory in application")
	}

	if meta != nil {
		if err := addGeneratedStartscript(scriptPath, libPath, *meta); err != nil {
			return errors.Wrap(err, "Failed to create default start script")
		}
	}

	if err := prepareEffectiveStartscript(scriptPath); err != nil {
		return errors.Wrap(err, "Failed to create effective start script")
	}

	return nil
}

func addGeneratedStartscript(scriptPath string, libPath string, meta config.DeliverableMetadata) error {

	classpath, err := Classpath(libPath)

	if err != nil {
		return errors.Wrap(err, "Failed to get application classpath")
	}

	startscript := NewStartscript(classpath, meta)

	if err = WriteFile(filepath.Join(scriptPath, "generated-start"), startscript); err != nil {
		return errors.Wrap(err, "Failed to write script")
	}

	return nil
}

func prepareEffectiveStartscript(scriptPath string) error {
	name := "os-start"

	defaultScriptExists, err := Exists(filepath.Join(scriptPath, name))

	if err != nil {
		return errors.Wrapf(err, "Could not determine if %s exists", name)
	} else if defaultScriptExists {
		return nil
	}

	for _, altScriptName := range []string{"os-start.sh", "generated-start", "start", "start.sh"} {
		scriptExists, err := Exists(filepath.Join(scriptPath, altScriptName))

		if err != nil {
			return errors.Wrapf(err, "Could not determine if %s exists", altScriptName)
		} else if !scriptExists {
			continue
		}

		if err := os.Symlink(altScriptName, filepath.Join(scriptPath, name)); err != nil {
			return errors.Wrapf(err, "Failed to create symlink %s to %s", altScriptName, name)
		}

		return nil
	}

	return fmt.Errorf("No start script found")
}

func Classpath(libPath string) ([]string, error) {

	files, err := ioutil.ReadDir(libPath)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to list directory")
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
			return "", errors.Wrapf(err, "Could not determine if directory %s exists", libdirPath)
		} else if exists {
			return libdirPath, nil
		}
	}

	return "", errors.Errorf("No lib directory found in application")
}
