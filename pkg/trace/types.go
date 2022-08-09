package trace

import (
	"github.com/skatteetaten/architect/v2/pkg/docker"
)

type BaseImage struct {
	Type        string                  `json:"type"`
	Name        string                  `json:"name"`
	Version     string                  `json:"version"`
	Digest      string                  `json:"digest"`
	ImageConfig *docker.ContainerConfig `json:"image_config"`
}

type DeployableImage struct {
	Type          string `json:"imageType"`
	Name          string `json:"name"`
	AppVersion    string `json:"appVersion"`
	AuroraVersion string `json:"auroraVersion"`
	Digest        string `json:"imageDigest"`
	Snapshot      bool   `json:"snapshot"`
	GitCommit     string `json:"gitCommit"`
}
