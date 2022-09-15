package format

import "github.com/anchore/syft/syft/artifact"

type PackageNode struct {
	PackageURL        string      `json:"purl,omitempty"`
	DependencyId      artifact.ID `json:"dependencyId,omitempty"`
	Name              string      `json:"name,omitempty"`
	Version           string      `json:"version,omitempty"`
	ChecksumAlgorithm string      `json:"checksumAlgorithm"`
	ChecksumValue     string      `json:"checksumValue"`
	SourceLocation    string      `json:"sourceLocation,omitempty"`
}
