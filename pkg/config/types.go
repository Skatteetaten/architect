package config

type ApplicationType string

const (
	JavaLeveransepakke   ApplicationType = "JavaLeveransepakke"
	NodeJsLeveransepakke ApplicationType = "NodeJsLeveranse"
)

type Config struct {
	ApplicationType ApplicationType
	MavenGav        *MavenGav
	NodeJSGav       *NodeJSGav
	Snapshot        bool //NPM don't have snapshot semantics, so this is lifted up.. Should be refactored
	DockerSpec      DockerSpec
	BuilderSpec     BuilderSpec
}

type MavenGav struct {
	ArtifactId string
	GroupId    string
	Version    string
	Classifier string
}

type NodeJSGav struct {
	NpmName string
	Version string
}

type DockerSpec struct {
	OutputRegistry   string
	OutputRepository string
	BaseImage        string
	BaseVersion      string
	PushExtraTags    PushExtraTags
	//This is the external docker registry where we check versions.
	ExternalDockerRegistry string
	TagWith                string
	RetagWith              string
	TagOverwrite           bool
}

//TODO Would it be more idiomatic Go to use bitmask?
type PushExtraTags struct {
	Latest bool
	Major  bool
	Minor  bool
	Patch  bool
}

type BuilderSpec struct {
	Version string
}
