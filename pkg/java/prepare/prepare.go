package prepare

import (
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	deliverable "github.com/skatteetaten/architect/pkg/java/config"
	"github.com/skatteetaten/architect/pkg/java/nexus"
	"github.com/skatteetaten/architect/pkg/java/prepare/resources"
	"github.com/skatteetaten/architect/pkg/util"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

func createEnv(auroraVersion *runtime.AuroraVersion, pushextratags config.PushExtraTags) map[string]string {
	env := make(map[string]string)
	env[docker.ENV_APP_VERSION] = string(auroraVersion.GetAppVersion())
	env[docker.ENV_AURORA_VERSION] = auroraVersion.GetCompleteVersion()
	env[docker.ENV_PUSH_EXTRA_TAGS] = pushextratags.ToStringValue()
	env[docker.TZ] = "Europe/Oslo"

	if auroraVersion.Snapshot {
		env[docker.ENV_SNAPSHOT_TAG] = auroraVersion.GetGivenVersion()
	}

	return env
}

type FileGenerator interface {
	Write(writer io.Writer) error
}

func Prepare(dockerSpec config.DockerSpec, auroraVersions *runtime.AuroraVersion, deliverable *nexus.Deliverable, baseImage *runtime.BaseImage) (string, error) {

	// Create docker build folder
	dockerBuildPath, err := ioutil.TempDir("", "deliverable")

	if err != nil {
		return "", errors.Wrap(err, "Failed to create root folder of Docker context")
	}

	// Unzip deliverable
	applicationFolder := filepath.Join(dockerBuildPath, ApplicationFolder)
	err = extractAndRenameDeliverable(dockerBuildPath, deliverable.Path)

	if err != nil {
		return "", errors.Wrap(err, "Failed to extract application archive")
	}

	// Load metadata
	meta, err := loadDeliverableMetadata(filepath.Join(applicationFolder, DeliveryMetadataPath))

	if err != nil {
		return "", errors.Wrap(err, "Failed to read application metadata")
	}

	// Runtime scripts
	if err := copyDefaultRuntimeScripts(dockerBuildPath); err != nil {
		return "", errors.Wrap(err, "Failed to add static content to Docker context")
	}

	// Prepare application
	if err := prepareEffectiveScripts(applicationFolder, meta); err != nil {
		return "", errors.Wrap(err, "Failed to prepare application")
	}

	// Dockerfile
	fileWriter := util.NewFileWriter(dockerBuildPath)

	if err = fileWriter(NewDockerfile(dockerSpec, auroraVersions, meta, baseImage), "Dockerfile"); err != nil {
		return "", errors.Wrap(err, "Failed to create Dockerfile")
	}

	return dockerBuildPath, nil
}

func extractAndRenameDeliverable(dockerBuildFolder string, deliverablePath string) error {

	applicationRoot := filepath.Join(dockerBuildFolder, ApplicationRoot)
	renamedApplicationFolder := filepath.Join(dockerBuildFolder, ApplicationFolder)
	if err := os.MkdirAll(dockerBuildFolder, 0755); err != nil {
		return errors.Wrap(err, "Failed to create application directory in Docker context")
	}

	if err := ExtractDeliverable(deliverablePath, applicationRoot); err != nil {
		return errors.Wrapf(err, "Failed to extract application archive")
	}

	if err := renameSingleFolderInDirectory(applicationRoot, renamedApplicationFolder); err != nil {
		return errors.Wrap(err, "Failed to rename application directory in Docker context")
	} else {
		return nil
	}
}

// When we unzip the delivery, it will have an additional level.
// eg. app/myapplication-LEVERANSEPAKKE-SNAPSHOT -> app/application
func renameSingleFolderInDirectory(base string, newName string) error {
	list, err := ioutil.ReadDir(base)

	if err != nil {
		return errors.Wrapf(err, "Failed to open application directory %s", base)
	} else if len(list) == 0 {
		return errors.Errorf("Application root folder does not exist %s", base)
	}

	folderToBeRenamed := filepath.Join(base, list[0].Name())
	if err = os.Rename(folderToBeRenamed, newName); err != nil {
		return errors.Wrapf(err, "Rename from %s to %s failed", folderToBeRenamed, newName)
	} else {
		return nil
	}
}

func loadDeliverableMetadata(metafile string) (*deliverable.DeliverableMetadata, error) {
	fileExists, err := Exists(metafile)

	if err != nil {
		return nil, errors.Wrapf(err, "Could not find %s in deliverable", path.Base(metafile))
	} else if !fileExists {
		return nil, errors.Errorf("Could not find %s in deliverable", path.Base(metafile))
	}

	reader, err := os.Open(metafile)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to open application metadata file")
	}

	deliverableMetadata, err := deliverable.NewDeliverableMetadata(reader)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to load metadata from %s", path.Base(metafile))
	}

	return deliverableMetadata, nil
}

func copyDefaultRuntimeScripts(dockerBuildPath string) error {
	scriptDirPath := filepath.Join(dockerBuildPath, "app", "architect")

	if err := os.MkdirAll(scriptDirPath, 0755); err != nil {
		return errors.Wrap(err, "Failed to create resource folder")
	}

	for _, asset := range resources.AssetNames() {
		bytes := resources.MustAsset(asset)
		err := ioutil.WriteFile(filepath.Join(scriptDirPath, asset), bytes, 0755)
		if err != nil {
			return errors.Wrapf(err, "Failed add resource %s", asset)
		}
	}

	return nil
}

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)

	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, errors.Wrap(err, "Failed to stat file")
	}

	return true, nil
}
