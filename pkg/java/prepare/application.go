package prepare

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/java/config"
	"github.com/skatteetaten/architect/pkg/util"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	DeliveryMetadataPath   = "metadata/openshift.json"
	DefaultLivenessScript  = "liveness_std.sh"
	DefaultReadinessScript = "readiness_std.sh"
)

func prepareEffectiveScripts(applicationPath string, meta *config.DeliverableMetadata) error {

	scriptPath := filepath.Join(applicationPath, "bin")

	if err := os.MkdirAll(scriptPath, 0755); err != nil {
		return errors.Wrap(err, "Failed to create directory")
	}

	libPath, err := findLibraryPath(applicationPath)

	if err != nil {
		return errors.Wrap(err, "Failed to locate lib directory in application")
	}

	fileWriter := util.NewFileWriter(scriptPath)

	ok, err := addGeneratedStartscript(fileWriter, applicationPath, libPath, meta)
	if err != nil {
		return errors.Wrap(err, "Failed to create default start script")
	}
	if !ok {
		logrus.Info("No metadata present for generating startscript")
	}

	if err := prepareEffectiveStartscript(scriptPath); err != nil {
		return errors.Wrap(err, "Failed to create effective start script")
	}

	if err := prepareLivelinessAndReadynessScripts(scriptPath); err != nil {
		return errors.Wrap(err, "Error preparing liveness and readiness scripts")
	}

	return nil
}

func addGeneratedStartscript(fileWriter util.FileWriter, applicationBase string, libPath string, meta *config.DeliverableMetadata) (bool, error) {

	if meta == nil || meta.Java == nil || len(meta.Java.MainClass) == 0 {
		return false, nil
	}

	classpath, err := generateClasspath(applicationBase, libPath)

	if err != nil {
		return false, errors.Wrap(err, "Failed to get application classpath")
	}

	if err = fileWriter(newStartScript(classpath, meta), "generated-start"); err != nil {
		return false, errors.Wrap(err, "Failed to write script")
	}

	return true, nil
}

func prepareLivelinessAndReadynessScripts(scriptPath string) error {
	livenessScript := filepath.Join(scriptPath, "liveness.sh")
	err := linkInDefaultIfNotExists(livenessScript,
		filepath.Join(DockerBasedir, "architect", DefaultLivenessScript))
	if err != nil {
		return errors.Wrap(err, "Could not link in script")
	}

	readinessScript := filepath.Join(scriptPath, "readiness.sh")
	err = linkInDefaultIfNotExists(readinessScript,
		filepath.Join(DockerBasedir, "architect", DefaultReadinessScript))
	if err != nil {
		return errors.Wrap(err, "Could not link in script")
	}
	return nil
}
func linkInDefaultIfNotExists(pathToScript string, pathToDefaultScript string) error {
	exists, err := Exists(pathToScript)
	if err != nil {
		return errors.Wrapf(err, "Could not determine if %s exists", pathToScript)
	}
	if !exists {
		logrus.Debugf("Did not find script %s, symlinking %s", pathToScript, pathToDefaultScript)
		err := os.Symlink(pathToDefaultScript, pathToScript)
		if err != nil {
			return errors.Wrap(err, "Error linking in script")
		}
	}
	return nil
}

func prepareEffectiveStartscript(scriptPath string) error {
	const name = "start"
	startScript := filepath.Join(scriptPath, "start")
	defaultScriptExists, err := Exists(startScript)
	if err != nil {
		return errors.Wrapf(err, "Could not determine if %s exists", name)
	}

	for _, altScriptName := range []string{"generated-start", "os-start.sh", "os-start", "start.sh"} {
		scriptExists, err := Exists(filepath.Join(scriptPath, altScriptName))

		if err != nil {
			return errors.Wrapf(err, "Could not determine if %s exists", altScriptName)
		} else if !scriptExists {
			continue
		}

		//If start exists, we rename it. It has precedence last
		if defaultScriptExists {
			err := os.Rename(startScript, filepath.Join(scriptPath, name+".bak"))
			if err != nil {
				return errors.Wrapf(err, "Unable to rename already present start-script")
			}
		}

		if err := os.Symlink(altScriptName, startScript); err != nil {
			return errors.Wrapf(err, "Failed to create symlink %s to %s", altScriptName, name)
		}

		return nil
	}

	//This has presedence after generated-start, os-start.sh and start.sh
	if defaultScriptExists {
		logrus.Debugf("Default startscript %s exists.", name)
		return nil
	}

	return errors.Errorf("No start script has been defined or generated for this Leveransepakke. %s",
		"Please put a script called one of start.sh or os-start.sh, in the bin folder, or fill inn correct "+
			"informastion in openshift.json")
}

// applicationDir is the temporary directory where we have the application code
// libpath is the path to the jar files
func generateClasspath(applicationDir string, libPath string) ([]string, error) {
	files, err := ioutil.ReadDir(libPath)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to list directory")
	}

	classpath := make([]string, len(files))

	base := "$HOME/" + ApplicationFolder + "/" + strings.TrimPrefix(libPath, applicationDir)
	for index, value := range files {
		classpath[index] = filepath.Join(base, value.Name())
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

	return "", errors.Errorf("No lib directory found in application in base %s", basedirPath)
}
