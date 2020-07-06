package docker

import (
	"encoding/json"
	"github.com/pkg/errors"
	"os"
)

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

type Layer struct {
	MediaType string `json:"mediaType"`
	Size      int    `json:"size"`
	Digest    string `json:"digest"`
}

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

//TODO: Shallow
func (m *ManifestV2) CleanCopy() *ManifestV2 {
	var layers []Layer
	copy(layers, m.Layers)

	return &ManifestV2{
		SchemaVersion: m.SchemaVersion,
		MediaType:     m.MediaType,
		Config:        m.Config,
		Layers:        layers,
	}
}
