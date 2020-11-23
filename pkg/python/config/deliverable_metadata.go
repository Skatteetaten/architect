package config

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
)

type DeliverableMetadata struct {
	Docker    *MetadataDocker    `json:"docker"`
	Python    *MetadataPython    `json:"python"`
	Openshift *MetadataOpenShift `json:"openshift"`
}

type MetadataDocker struct {
	Maintainer string            `json:"maintainer"`
	Labels     map[string]string `json:"labels"`
}

type MetadataPython struct {
}

type MetadataOpenShift struct {
	ReadinessURL              string `json:"readinessUrl"`
	ReadinessOnManagementPort string `json:"readinessOnManagementPort"`
}

func NewDeliverableMetadata(reader io.Reader) (*DeliverableMetadata, error) {
	var meta DeliverableMetadata
	content, err := ioutil.ReadAll(reader)

	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(content, &meta); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal json metadata")
	}

	return &meta, nil
}
