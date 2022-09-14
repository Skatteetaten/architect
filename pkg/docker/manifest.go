package docker

import (
	"encoding/json"
	"github.com/pkg/errors"
	"os"
)

// ManifestV2 is the go representation of a docker manifest
type ManifestV2 struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"config"`
	Layers []Layer `json:"layers"`
}

// Layer represents a container layer
type Layer struct {
	MediaType string `json:"mediaType"`
	Size      int    `json:"size"`
	Digest    string `json:"digest"`
}

// Save Write the manifest to file
func (m *ManifestV2) Save(dstFolder string, name string) error {
	manifestFile, err := os.Create(dstFolder + "/" + name)
	if err != nil {
		return err
	}
	defer manifestFile.Close()

	data, err := json.Marshal(m)
	if err != nil {
		return errors.Wrap(err, "Manifest save: Marshal failed")
	}

	_, err = manifestFile.Write(data)
	if err != nil {
		return errors.Wrap(err, "Manifest save: failed")
	}

	return nil
}

// CleanCopy make a clean copy of the manifest
func (m *ManifestV2) CleanCopy() *ManifestV2 {
	layers := make([]Layer, len(m.Layers))
	copy(layers, m.Layers)
	return &ManifestV2{
		SchemaVersion: m.SchemaVersion,
		MediaType:     m.MediaType,
		Config:        m.Config,
		Layers:        layers,
	}
}
