package prepare

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/java/config"
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

func PrepareApplication(applicationPath string, applicationBase string, meta *config.DeliverableMetadata) error {

	scriptPath := filepath.Join(applicationPath, "bin")

	if err := os.MkdirAll(scriptPath, 0755); err != nil {
		return errors.Wrap(err, "Failed to create directory")
	}

	libPath, err := findLibraryPath(applicationPath)

	if err != nil {
		return errors.Wrap(err, "Failed to locate lib directory in application")
	}

	if meta != nil {
		if err := addGeneratedStartscript(scriptPath, applicationBase, libPath, *meta); err != nil {
			return errors.Wrap(err, "Failed to create default start script")
		}
	}

	if err := prepareEffectiveStartscript(scriptPath); err != nil {
		return errors.Wrap(err, "Failed to create effective start script")
	}

	if err := prepareLivelinessAndReadynessScripts(scriptPath); err != nil {
		return errors.Wrap(err, "Error preparing liveness and readiness scripts")
	}

	return nil
}

func addGeneratedStartscript(scriptPath string, applicationBase string, libPath string, meta config.DeliverableMetadata) error {

	classpath, err := Classpath(applicationBase, libPath)

	if err != nil {
		return errors.Wrap(err, "Failed to get application classpath")
	}

	startscript := NewStartscript(classpath, meta)

	if err = WriteFile(filepath.Join(scriptPath, "generated-start"), startscript); err != nil {
		return errors.Wrap(err, "Failed to write script")
	}

	return nil
}

func prepareLivelinessAndReadynessScripts(scriptPath string) error {
	livenessScript := filepath.Join(scriptPath, "liveness.sh")
	livelinessExists, err := Exists(livenessScript)
	if err != nil {
		return errors.Wrap(err, "Could not determine if liveness-script exists")
	}

	if !livelinessExists {
		logrus.Info("No liveness-script. Linking in default")
		err := os.Symlink(filepath.Join(DockerBasedir, "bin", DefaultLivenessScript), livenessScript)
		if err != nil {
			return errors.Wrap(err, "Error linking in script")
		}
	}

	readinessScript := filepath.Join(scriptPath, "readiness.sh")
	readinessExists, err := Exists(readinessScript)
	if err != nil {
		return errors.Wrap(err, "Could not determine if liveness-script exists")
	}

	if !readinessExists {
		logrus.Info("No readyness-script. Linking in default")
		err := os.Symlink(filepath.Join(DockerBasedir, "bin", DefaultReadinessScript), readinessScript)
		if err != nil {
			return errors.Wrap(err, "Error linking in script")
		}

	}

	return err
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

	return errors.Errorf("No start script has been defined or generated for this Leveransepakke. %s",
		"Please put a script called one of start, start.sh, os-start or os-start.sh in the bin folder.")
}

// applicationDir is the temporary directory where we have the application code
// libpath is the path to the jar files
func Classpath(applicationDir string, libPath string) ([]string, error) {
	files, err := ioutil.ReadDir(libPath)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to list directory")
	}

	classpath := make([]string, len(files))

	base := DockerBasedir + strings.TrimPrefix(libPath, applicationDir)
	logrus.Info(base)
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

	return "", errors.Errorf("No lib directory found in application")
}
