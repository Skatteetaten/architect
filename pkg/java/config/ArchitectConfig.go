package config

import "encoding/json"

type ArchitectConfig struct {
	Docker *struct {
		Maintainer string `json:"maintainer"`
		Labels     interface{} `json:"labels"`
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

func NewFromJson(input string) *ArchitectConfig {
	var cfg ArchitectConfig
	json.Unmarshal([]byte(input), &cfg)
	return &cfg
}
