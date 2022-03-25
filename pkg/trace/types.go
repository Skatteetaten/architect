package trace

import (
	"github.com/skatteetaten/architect/v2/pkg/docker"
	"github.com/skatteetaten/architect/v2/pkg/nexus"
)

type BaseImage struct {
	Type        string                  `json:"type"`
	Name        string                  `json:"name"`
	Version     string                  `json:"version"`
	Digest      string                  `json:"digest"`
	ImageConfig *docker.ContainerConfig `json:"image_config"`
}

type DeployableImage struct {
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Digest       string                 `json:"digest"`
	Tags         map[string]string      `json:"tags"`
	NexusSHA1    string                 `json:"nexus_sha1"`
	ImageConfig  map[string]interface{} `json:"image_config"`
	Dependencies []nexus.Dependency     `json:"dependencies"`
}
