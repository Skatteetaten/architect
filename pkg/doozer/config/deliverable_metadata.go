package config

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io"
)

// DeliverableMetadata build and runtime configuration
type DeliverableMetadata struct {
	Docker    *MetadataDocker    `json:"docker"`
	Doozer    *MetadataDoozer    `json:"doozer"`
	Java      *MetadataJava      `json:"java"`
	Openshift *MetadataOpenShift `json:"openshift"`
}

// MetadataDocker maintainer and labels. These values are appended to the resulting image.
type MetadataDocker struct {
	Maintainer string            `json:"maintainer"`
	Labels     map[string]string `json:"labels"`
}

// MetadataDoozer build specific information for dozer builds.
type MetadataDoozer struct {
	SrcPath      string `json:"srcPath"`
	FileName     string `json:"fileName"`
	DestPath     string `json:"destPath"`
	DestFilename string `json:"destFilename"` // Optional. Will use FileName as default
	CmdScript    string `json:"cmdScript"`    // Optional if base image CMD is applicable
	Entrypoint   string `json:"entrypoint"`   // Optional if base image Entrypoint is applicable
}

// MetadataJava java runtime information
type MetadataJava struct {
	MainClass       string `json:"mainClass"`
	JvmOpts         string `json:"jvmOpts"`
	ApplicationArgs string `json:"applicationArgs"`
}

// MetadataOpenShift readiness check parameters
type MetadataOpenShift struct {
	ReadinessURL              string `json:"readinessUrl"`
	ReadinessOnManagementPort string `json:"readinessOnManagementPort"`
}

// NewDeliverableMetadata read openshift.json and transform to DeliverableMetadata
func NewDeliverableMetadata(reader io.Reader) (*DeliverableMetadata, error) {
	var meta DeliverableMetadata
	content, err := io.ReadAll(reader)

	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(content, &meta); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal json metadata")
	}

	return &meta, nil
}
