package config

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
)

type DeliverableMetadata struct {
	Docker    *MetadataDocker    `json:"docker"`
	Doozer    *MetadataDoozer    `json:"doozer"`
	Java      *MetadataJava      `json:"java"`
	Openshift *MetadataOpenShift `json:"openshift"`
}

type MetadataDocker struct {
	Maintainer string            `json:"maintainer"`
	Labels     map[string]string `json:"labels"`
}

// TODO: Consider if "destPath" is available for fetching from image in some way and can be optional
type MetadataDoozer struct {
	SrcPath      string `json:"srcPath"`
	FileName     string `json:"fileName"`
	DestPath     string `json:"destPath"`
	DestFilename string `json:"destFilename"`
	CmdScript    string `json:"cmdScript"`
}

// TODO: Consider if MetadataJava is at all necessary for doozer builds
type MetadataJava struct {
	MainClass       string `json:"mainClass"`
	JvmOpts         string `json:"jvmOpts"`
	ApplicationArgs string `json:"applicationArgs"`
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
