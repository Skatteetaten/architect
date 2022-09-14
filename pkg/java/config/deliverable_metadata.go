package config

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io"
)

// DeliverableMetadata build and runtime configuration.
// The information is written to the radish descriptor file [radish.json]
// in the resulting image.
type DeliverableMetadata struct {
	Docker    *MetadataDocker    `json:"docker"`
	Java      *MetadataJava      `json:"java"`
	Openshift *MetadataOpenShift `json:"openshift"`
}

// MetadataDocker maintainer and labels. These values are appended to the resulting image.
type MetadataDocker struct {
	Maintainer  string            `json:"maintainer"`
	Labels      map[string]string `json:"labels"`
	BaseImage   string            `json:"baseImage"`
	BaseVersion string            `json:"baseVersion"`
}

// MetadataJava java runtime configuration
type MetadataJava struct {
	MainClass       string `json:"mainClass"`
	JvmOpts         string `json:"jvmOpts"`
	ApplicationArgs string `json:"applicationArgs"`
	ReadinessURL    string `json:"readinessUrl"`
	StartScript     string `json:"startScript"`
}

// MetadataOpenShift readiness parameters
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
