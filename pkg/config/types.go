package config

import (
	"strings"
	"time"
)

type ApplicationType string

const (
	JavaLeveransepakke   ApplicationType = "JavaLeveransepakke"
	NodeJsLeveransepakke ApplicationType = "NodeJsLeveranse"
	DoozerLeveranse      ApplicationType = "DoozerLeveranse"
	PythonLeveranse      ApplicationType = "PythonLeveranse"
	NodeJs                               = "NODEJS"
	Doozer                               = "DOOZER"
	Python                               = "PYTHON"
)

type PackageType string

const (
	ZipPackaging PackageType = "zip"
	TgzPackaging PackageType = "tgz"
)

type Classifier string

const (
	Leveransepakke       Classifier = "Leveransepakke"
	Webleveransepakke    Classifier = "Webleveransepakke"
	Doozerleveransepakke Classifier = "Doozerleveransepakke"
	Pythonleveransepakke Classifier = "Pythonleveransepakke"
)

const (
	Docker  = "docker"
	Buildah = "buildah"
)

type Config struct {
	ApplicationType   ApplicationType
	ApplicationSpec   ApplicationSpec
	DockerSpec        DockerSpec
	BuilderSpec       BuilderSpec
	NexusAccess       NexusAccess
	BinaryBuild       bool
	LocalBuild        bool
	BuildStrategy     string
	TlsVerify         bool
	BuildTimeout      time.Duration
	NoPush            bool
	SporingsContext   string
	Sporingstjeneste  string
	OwnerReferenceUid string
}

type NexusAccess struct {
	Username string
	Password string
	NexusUrl string
}

func (n NexusAccess) String() string {
	return "{Username:" + n.Username + " Password:****** NexusUrl:" + n.NexusUrl + "}"
}

type ApplicationSpec struct {
	MavenGav      MavenGav
	BaseImageSpec DockerBaseImageSpec
}

type MavenGav struct {
	ArtifactId string
	GroupId    string
	Version    string
	Classifier Classifier
	Type       PackageType
}

func (m *MavenGav) IsSnapshot() bool {
	return strings.Contains(m.Version, "SNAPSHOT")
}

func (m *MavenGav) Name() string {
	return strings.Join([]string{m.GroupId, m.ArtifactId, m.ArtifactId}, ":")
}

type DockerBaseImageSpec struct {
	BaseImage   string
	BaseVersion string
}

type DockerSpec struct {
	OutputRegistry       string
	OutputRepository     string
	InternalPullRegistry string
	PushExtraTags        PushExtraTags
	//This is the external docker registry where we check versions.
	ExternalDockerRegistry string
	//The tag to push to. This is only used for ImageStreamTags (as for now) and RETAG functionality
	TagWith      string
	RetagWith    string
	TagOverwrite bool
}

type BuilderSpec struct {
	Version string
}

type PushExtraTags struct {
	Latest bool
	Major  bool
	Minor  bool
	Patch  bool
}

// Generates the tags given the appversion and extra tag configuration. Don't do any filtering
func (m *PushExtraTags) ToStringValue() string {
	str := make([]string, 0, 5)
	if m.Major {
		str = append(str, "major")
	}
	if m.Minor {
		str = append(str, "minor")
	}
	if m.Patch {
		str = append(str, "patch")
	}
	if m.Latest {
		str = append(str, "latest")
	}
	return strings.Join(str, ",")
}

func (m DockerSpec) GetExternalRegistryWithoutProtocol() string {
	return strings.TrimPrefix(m.ExternalDockerRegistry, "https://")
}

func (m DockerSpec) GetInternalPullRegistryWithoutProtocol() string {
	return strings.TrimPrefix(m.InternalPullRegistry, "https://")
}

func ParseExtraTags(i string) PushExtraTags {
	p := PushExtraTags{}
	if strings.Contains(i, "major") {
		p.Major = true
	}
	if strings.Contains(i, "minor") {
		p.Minor = true
	}
	if strings.Contains(i, "patch") {
		p.Patch = true
	}
	if strings.Contains(i, "latest") {
		p.Latest = true
	}
	return p
}
