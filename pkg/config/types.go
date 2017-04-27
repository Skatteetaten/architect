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
	BuilderSpec	BuilderSpec
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
	BaseVersion	 string
	PushExtraTags	 string
	//This is the external docker registry where we check versions.
	ExternalDockerRegistry string
}

type BuilderSpec struct {
	Version		string
}

type Deliverable struct {
	Path 		string
}
