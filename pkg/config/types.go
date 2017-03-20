package config

type ApplicationType string

const (
	JavaLeveransepakke ApplicationType = "JavaLeveransepakke"
	NodeJsLeveransepakke ApplicationType = "NodeJsLeveranse"
)

type Config struct {
	ApplicationType ApplicationType
	NexusGav        NexusGav
	DockerSpec      DockerSpec
}

type NexusGav struct {
	ArtifactId string
	GroupId    string
	Version    string
	Classifier string
}

type DockerSpec struct {
	Registry string
}
