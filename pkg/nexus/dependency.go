package nexus

import (
	"fmt"
	"os"
	"path/filepath"
)

func ExtractDependecyMetadata(buildFolder string) ([]Dependency, error) {

	metadata := []Dependency{}
	fileList := []string{}
	err := filepath.Walk(buildFolder, func(path string, f os.FileInfo, err error) error {
		fileList = append(fileList, path)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("Failed when traversing folder %s", buildFolder)
	}

	for _, file := range fileList {
		info, err := os.Stat(file)
		if err == nil && info.IsDir() {
			continue
		}
		sha1, _ := hashFileSHA1(file)
		if sha1 == "" {
			continue
		}
		metadata = append(metadata, Dependency{
			Name: filepath.Base(file),
			SHA1: sha1,
			Size: info.Size(),
		})
	}

	return metadata, nil
}
