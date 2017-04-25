package config

type ApplicationType string

const (
	JavaLeveransepakke ApplicationType = "JavaLeveransepakke"
	NodeJsLeveransepakke ApplicationType = "NodeJsLeveranse"
)

type Config struct {
	ApplicationType ApplicationType
	MavenGav        MavenGav
	DockerSpec      DockerSpec
}

type MavenGav struct {
	ArtifactId string
	GroupId    string
	Version    string
	Classifier string
}

type DockerSpec struct {
	OutputRegistry   string
	OutputRepository string
	BaseImage        string
}
