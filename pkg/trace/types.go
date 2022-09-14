package trace

import (
	"github.com/skatteetaten/architect/v2/pkg/docker"
)

// BaseImage representation used in trace
type BaseImage struct {
	Type        string                  `json:"type"`
	Name        string                  `json:"name"`
	Version     string                  `json:"version"`
	Digest      string                  `json:"digest"`
	ImageConfig *docker.ContainerConfig `json:"image_config"`
}

// DeployableImage representation used in trace
type DeployableImage struct {
	Type             string `json:"imageType"`
	Name             string `json:"name"`
	AppVersion       string `json:"appVersion"`
	Digest           string `json:"imageDigest"`
	Snapshot         bool   `json:"snapshot"`
	GitCommit        string `json:"gitCommit"`
	BaseImageName    string `json:"baseImageName"`
	BaseImageVersion string `json:"baseImageVersion"`
	BaseImageDigest  string `json:"baseImageDigest"`
	BuildVersion     string `json:"buildVersion"`
}
