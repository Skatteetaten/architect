package config

import ("encoding/json"
	"strings"
	"fmt"
)

type ArchitectConfig interface {
	GetMaintainer() (string)
	GetLabels() (string)
	GetReadinessEnv() (string)
}

type Root struct {
	Docker *struct {
		Maintainer string `json:"maintainer"`
		Labels interface{} `json:"labels"`
	} `json:"docker"`
	Java *struct {
		MainClass string `json:"mainClass"`
		JvmOpts string `json:"jvmOpts"`
		ApplicationArgs string `json:"applicationArgs"`
		ReadinessURL string `json:"readinessUrl"`
	} `json:"java"`
	Openshift *struct {
		ReadinessURL string `json:"readinessUrl"`
		ReadinessOnManagementPort string `json:"readinessOnManagementPort"`
	} `json:"openshift"`
}

func CreateDefaultImpl(input string) *ArchitectConfig {
	var impl *JsonConfig = &JsonConfig{}
	impl.init(input)
	var iconf ArchitectConfig = impl

	return &iconf
}

type JsonConfig struct {
	maintainer string
	labels string
	readinessEnv string
}

func (cfg *JsonConfig) GetMaintainer() (string) {
	return cfg.maintainer
}

func (cfg *JsonConfig) GetLabels() (string) {
	return cfg.labels
}

func (cfg *JsonConfig) GetReadinessEnv() (string) {
	return cfg.readinessEnv
}

func joinMap(m map[string]interface{}) string {
	var labels []string

	for k, v := range m {
		labels = append(labels, fmt.Sprintf("%s=\"%s\"", k, v))
	}

	return strings.Join(labels, " ")
}

func (cfg *JsonConfig) init(input string) (error) {
	var root Root

	json.Unmarshal([]byte(input), &root)

	cfg.maintainer = root.Docker.Maintainer

	if root.Docker.Labels == nil {
		cfg.labels = ""
	} else {
		cfg.labels = joinMap(root.Docker.Labels.(map[string]interface{}))
	}

	m := make(map[string]interface{})

	if root.Openshift != nil {
		if root.Openshift.ReadinessURL != "" {
			m["READINESS_CHECK_URL"] = root.Openshift.ReadinessURL
		}

		if root.Openshift.ReadinessOnManagementPort == "" || root.Openshift.ReadinessOnManagementPort == "true" {
			m["READINESS_ON_MANAGEMENT_PORT"] = "true"
		}
	} else if root.Java != nil && root.Java.ReadinessURL != "" {
		m["READINESS_CHECK_URL"] = root.Java.ReadinessURL
	}

	cfg.readinessEnv = joinMap(m)

	return nil
}