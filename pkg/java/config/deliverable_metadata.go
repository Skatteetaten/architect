package config

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
)

type DeliverableMetadata struct {
	Docker *struct {
		Maintainer string            `json:"maintainer"`
		Labels     map[string]string `json:"labels"`
	} `json:"docker"`
	Java *struct {
		MainClass       string `json:"mainClass"`
		JvmOpts         string `json:"jvmOpts"`
		ApplicationArgs string `json:"applicationArgs"`
		ReadinessURL    string `json:"readinessUrl"`
	} `json:"java"`
	Openshift *struct {
		ReadinessURL              string `json:"readinessUrl"`
		ReadinessOnManagementPort string `json:"readinessOnManagementPort"`
	} `json:"openshift"`
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
