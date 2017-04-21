package prepare

import (
	"bufio"
	"fmt"
	"github.com/skatteetaten/architect/pkg/java/config"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"github.com/skatteetaten/architect/pkg/docker"
	"strings"
)

const (
	ScriptSrcPath = "resources" // Temporary, untill a solution is found for distribution of the scripts.
)

type FileGenerator interface {
	Write(writer io.Writer) error
}

func Prepare(baseImage string, env map[string]string, deliverablePath string) (string, error) {

	// Create docker build folder
	dockerBuildPath, err := ioutil.TempDir("", "deliverable")

	if err != nil {
		return "", fmt.Errorf("Failed to create root folder: %v", err)
	}

	// Unzip deliverable
	applicationPath, err := extractDeliverable(dockerBuildPath, deliverablePath)

	if err != nil {
		return "", fmt.Errorf("Failed to unzip deliverable: %v", err)
	}

	// Load metadata
	meta, err := loadDeliverableMetadata(filepath.Join(applicationPath, DeliveryMetadataPath))

	if err != nil {
		return "", fmt.Errorf("Failed to load deliverable metadata: %v", err)
	}

	// Prepare application
	if err := PrepareApplication(applicationPath, meta); err != nil {
		return "", fmt.Errorf("Failed to prepare application: %v", err)
	}

	// Dockerfile
	if err = addDockerfile(dockerBuildPath, meta, baseImage, env); err != nil {
		return "", fmt.Errorf("Failed to create dockerfile: %v", err)
	}

	// Runtime scripts
	if err := addRuntimeScripts(ScriptSrcPath, dockerBuildPath); err != nil {
		return "", fmt.Errorf("Failed to add scripts: %v", err)
	}

	return dockerBuildPath, nil
}

func extractDeliverable(dockerBuildPath string, deliverablePath string) (string, error) {
	applicationBase := filepath.Join(dockerBuildPath, "app")
	applicationPath := filepath.Join(applicationBase, "application")

	if err := os.MkdirAll(applicationBase, 0755); err != nil {
		return "", err
	}

	if err := ExtractDeliverable(deliverablePath, filepath.Join(dockerBuildPath, "app")); err != nil {
		return "", fmt.Errorf("Failed to unzip deliverable %s: %v", deliverablePath, err)
	}

	if err := renameApplicationDir(applicationBase); err != nil {
		return "", err
	}

	return applicationPath, nil
}

func renameApplicationDir(base string) error {
	list, err := ioutil.ReadDir(base)

	if err != nil {
		return err
	} else if len(list) == 0 {
		return fmt.Errorf("Root folder does not exist")
	}

	return os.Rename(filepath.Join(base, list[0].Name()), filepath.Join(base, "application"))
}

func addDockerfile(basedirPath string, meta *config.DeliverableMetadata, baseImage string, env map[string]string) error {
	baseImage, err := fixBaseImageTag(baseImage)

	if err != nil {
		return err
	}

	dockerfile := NewDockerfile(baseImage, env, meta)

	if err := WriteFile(filepath.Join(basedirPath, "Dockerfile"), dockerfile); err != nil {
		return err
	}

	return nil
}

func fixBaseImageTag(baseImage string) (string, error) {
	repo, tag, err := FindRepoAndTagFromBaseImage(baseImage, ":")

	if err != nil {
		return "", err
	}

	rc := docker.NewRegistryClient("http://uil0map-paas-app01:9090") // TODO Handle registry address

	biv, err := docker.GetManifestEnv(*rc, repo, tag, "BASE_IMAGE_VERSION")

	if err != nil {
		return "", err
	} else if biv == "" {
		return "", fmt.Errorf("Failed to get base image version from Docker registry")
	}

	return fmt.Sprintf("%s:%s", repo, biv), nil
}

func loadDeliverableMetadata(path string) (*config.DeliverableMetadata, error) {
	fileExists, err := Exists(path)

	if err != nil {
		return nil, err
	} else if !fileExists {
		return nil, nil
	}

	reader, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	deliverableMetadata, err := config.NewDeliverableMetadata(reader)

	if err != nil {
		return nil, err
	}

	return deliverableMetadata, nil
}

func addRuntimeScripts(ScriptSrcPath, dockerBuildPath string) error {
	scriptDirPath := filepath.Join(dockerBuildPath, "app", "bin")

	if err := os.MkdirAll(scriptDirPath, 0755); err != nil {
		return err
	}

	for _, script := range []string{"run", "run_tools.sh", "readiness_std.sh", "liveness_std.sh"} {
		if err := Copy(filepath.Join(ScriptSrcPath, script), filepath.Join(scriptDirPath, script)); err != nil {
			return err
		}
	}

	return nil
}

func WriteFile(path string, generator FileGenerator) error {

	file, err := os.Create(path)

	if err != nil {
		return err
	}

	defer file.Close()

	writer := bufio.NewWriter(file)

	err = generator.Write(writer)

	if err != nil {
		return err
	}

	return writer.Flush()
}

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)

	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func Copy(srcPath, dstPath string) error {

	srcFile, err := os.Open(srcPath)

	if err != nil {
		return err
	}

	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)

	if err != nil {
		return err
	}

	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)

	closeErr := dstFile.Close()

	if err != nil {
		return err
	}

	return closeErr
}

func FindRepoAndTagFromBaseImage (target string, sep string) (string, string, error) {
	s := strings.Split(target, sep)

	if len(s) < 2 {
		return "", "", fmt.Errorf("Invalid base image reference: %s", target)
	}

	return strings.Join(s[:len(s)-1], sep), s[len(s)-1], nil
}
