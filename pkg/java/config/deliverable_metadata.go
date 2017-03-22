package config

import "encoding/json"

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

func NewDeliverableMetadata(input string) (*DeliverableMetadata, error) {
	var meta DeliverableMetadata
	err := json.Unmarshal([]byte(input), &meta)

	return &meta, err
}
